package options

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/internal/collections"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	DefaultRetryCount              = 3
	DefaultRestartDurationSeconds  = 60
	DefaultCMSQueryIntervalSeconds = 10
)

var AvailabilityModes = []string{"strong", "weak", "force"}

type StartedTime struct {
	Timestamp time.Time
	Direction rune
}

type VersionSpec struct {
	Sign  string
	Major int
	Minor int
	Patch int
}

type RestartOptions struct {
	AvailabilityMode   string
	Hosts              []string
	ExcludeHosts       []string
	RestartDuration    int
	RestartRetryNumber int
	Version            string
	CMSQueryInterval   int

	StartedTime *StartedTime
	VersionSpec *VersionSpec

	Continue bool

	*TenantOptions
}

var (
	startedUnparsedFlag string
	versionUnparsedFlag string
)

var RestartOptionsInstance = &RestartOptions{ 
	TenantOptions: &TenantOptions{},
}

func (o *RestartOptions) Validate() error {
	if !collections.Contains(AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if o.RestartDuration < 0 {
		return fmt.Errorf("specified invalid restart duration seconds: %d. Must be positive", o.RestartDuration)
	}

	if o.CMSQueryInterval < 0 {
		return fmt.Errorf("specified invalid cms query interval seconds: %d. Must be positive", o.RestartDuration)
	}

	if o.RestartRetryNumber < 0 {
		return fmt.Errorf("specified invalid restart retry number: %d. Must be positive", o.RestartRetryNumber)
	}

	if startedUnparsedFlag != "" {
		directionRune := []rune(startedUnparsedFlag)[0]
		if directionRune != '<' && directionRune != '>' {
			return fmt.Errorf("the first character of --started value should be < or >.")
		}

		timestampString, _ := strings.CutPrefix(startedUnparsedFlag, string(directionRune))
		timestamp, err := time.Parse(time.RFC3339, timestampString)
		if err != nil {
			return fmt.Errorf("failed to parse --started: %w", err)
		}

		o.StartedTime = &StartedTime{
			Timestamp: timestamp,
			Direction: directionRune,
		}
	}

	if versionUnparsedFlag != "" {
		pattern := `^(>|<|!=|==)(\d+|\*)\.(\d+|\*)\.(\d+|\*)$`
		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(versionUnparsedFlag)
		if len(matches) == 5 {
			major, _ := strconv.Atoi(matches[2])
			minor, _ := strconv.Atoi(matches[3])
			patch, _ := strconv.Atoi(matches[4])
			o.VersionSpec = &VersionSpec{
				Sign:  matches[1],
				Major: major,
				Minor: minor,
				Patch: patch,
			}
		} else {
			return fmt.Errorf("failed to parse --version flag. %s value does not satisfy the format, check --help", versionUnparsedFlag)
		}
	}

	_, errFromIds := o.GetNodeIds()
	_, errFromFQDNs := o.GetNodeFQDNs()
	if errFromIds != nil && errFromFQDNs != nil {
		return fmt.Errorf(
			"failed to parse --hosts argument as node ids (%w) or host fqdns (%w)",
			errFromIds,
			errFromFQDNs,
		)
	}

	return nil
}

func (o *RestartOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Hosts, "hosts", o.Hosts,
		`Restart only specified hosts. You can specify a list of host FQDNs or a list of node ids, 
but you can not mix host FQDNs and node ids in this option. The list is comma-delimited.
  E.g.: '--hosts 1,2,3' or '--hosts fqdn1,fqdn2,fqdn3'`)

	fs.StringSliceVar(&o.ExcludeHosts, "exclude-hosts", []string{},
		`Comma-delimited list. Do not restart these hosts, even if they are explicitly specified in --hosts.`)

	fs.StringVar(&o.AvailabilityMode, "availability-mode", "strong",
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(AvailabilityModes, ", ")))

	fs.IntVar(&o.RestartDuration, "restart-duration", DefaultRestartDurationSeconds,
		`CMS will release the node for maintenance for restart-duration * restart-retry-number seconds. Any maintenance
after that would be considered a regular cluster failure`)

	fs.IntVar(&o.RestartRetryNumber, "restart-retry-number", DefaultRetryCount,
		fmt.Sprintf("How many times a node should be retried on error, default %v", DefaultRetryCount))

	fs.IntVar(&o.CMSQueryInterval, "cms-query-interval", DefaultCMSQueryIntervalSeconds,
		fmt.Sprintf("How often to query CMS while waiting for new permissions %v", DefaultCMSQueryIntervalSeconds))

	fs.StringVar(&startedUnparsedFlag, "started", "",
		fmt.Sprintf("Apply filter by node started time. Format: [<>%%Y-%%m-%%dT%%H:%%M:%%SZ], e.g. >2024-03-13T17:20:06Z"))

	fs.StringVar(&versionUnparsedFlag, "version", "",
		`Apply filter by node version. Format: [<|>|!=|==MAJOR.MINOR.PATCH], e.g. '--version !=24.1.2'`)

	fs.BoolVar(&o.Continue, "continue", false,
		`Attempt to continue previous rolling restart, if there was one. The set of selected nodes 
for this invocation must be the same as for the previous invocation, and this can not be verified at runtime since 
the ydbops utility is stateless. Use at your own risk.`)
}

func (o *RestartOptions) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}

func (o *RestartOptions) GetRestartDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.RestartDuration) * time.Duration(o.RestartRetryNumber))
}

func (o *RestartOptions) GetNodeFQDNs() ([]string, error) {
	hosts := make([]string, 0, len(o.Hosts))

	for _, hostFqdn := range o.Hosts {
		_, err := url.Parse(hostFqdn)
		if err != nil {
			return nil, fmt.Errorf("invalid host fqdn specified: %s", hostFqdn)
		}

		hosts = append(hosts, hostFqdn)
	}

	return hosts, nil
}

func (o *RestartOptions) GetNodeIds() ([]uint32, error) {
	ids := make([]uint32, 0, len(o.Hosts))

	for _, nodeId := range o.Hosts {
		id, err := strconv.Atoi(nodeId)
		if err != nil {
			return nil, fmt.Errorf("failed to parse node id: %+v", err)
		}
		if id < 0 {
			return nil, fmt.Errorf("invalid node id specified: %d, must be positive", id)
		}
		ids = append(ids, uint32(id))
	}

	return ids, nil
}
