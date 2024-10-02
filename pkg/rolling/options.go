package rolling

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/profile"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	DefaultRetryCount              = 3
	DefaultRestartDurationSeconds  = 60
	DefaultCMSQueryIntervalSeconds = 10
)

var (
	majorMinorPatchPattern = `^(>|<|!=|~=)(\d+|\*)\.(\d+|\*)\.(\d+|\*)$`
	majorMinorPatchRegexp  = regexp.MustCompile(majorMinorPatchPattern)

	rawPattern = `^(==|!=)(.*)$`
	rawRegexp  = regexp.MustCompile(rawPattern)
)

type RestartOptions struct {
	AvailabilityMode           string
	Datacenters                []string
	Hosts                      []string
	ExcludeHosts               []string
	RestartDuration            int
	RestartRetryNumber         int
	Version                    string
	CMSQueryInterval           int
	SuppressCompatibilityCheck bool

	StartedTime *options.StartedTime
	VersionSpec options.VersionSpec

	Continue bool

	Storage    bool
	Tenant     bool
	TenantList []string

	SSHArgs []string

	CustomSystemdUnitName string

	KubeconfigPath string
	K8sNamespace   string

	MaxStaticNodeId int
}

var (
	startedUnparsedFlag string
	versionUnparsedFlag string
	rawSSHUnparsedArgs  string
)

func (o *RestartOptions) Validate() error {
	if !collections.Contains(options.AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if len(o.KubeconfigPath) > 0 && len(o.K8sNamespace) == 0 {
		return fmt.Errorf("specified --kubeconfig, but not --k8s-namespace")
	}

	if o.MaxStaticNodeId < 0 {
		return fmt.Errorf("specified invalid max-static-node-id: %d. Must be positive", o.MaxStaticNodeId)
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

	if len(o.TenantList) > 0 && !o.Tenant {
		return fmt.Errorf("--tenant-list specified, but --tenant is not explicitly specified." +
			"Please specify --tenant as well to clearly indicate your intentions.")
	}

	if startedUnparsedFlag != "" {
		directionRune, _ := utf8.DecodeRuneInString(startedUnparsedFlag)
		if directionRune != '<' && directionRune != '>' {
			return fmt.Errorf("the first character of --started value should be < or >")
		}

		timestampString, _ := strings.CutPrefix(startedUnparsedFlag, string(directionRune))
		timestamp, err := time.Parse(time.RFC3339, timestampString)
		if err != nil {
			return fmt.Errorf("failed to parse --started: %w", err)
		}

		o.StartedTime = &options.StartedTime{
			Timestamp: timestamp,
			Direction: directionRune,
		}
	}

	if versionUnparsedFlag != "" {
		var err error
		o.VersionSpec, err = parseVersionFlag(versionUnparsedFlag)
		if err != nil {
			return err
		}
	}

	o.SSHArgs = utils.ParseSSHArgs(rawSSHUnparsedArgs)

	_, errFromIds := utils.GetNodeIds(o.Hosts)
	_, errFromFQDNs := utils.GetNodeFQDNs(o.Hosts)
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
	fs.BoolVar(&o.Storage, "storage", false, `Only include storage nodes. Otherwise, include all nodes by default`)

	fs.BoolVar(&o.Tenant, "tenant", false, `Only include tenant nodes. Otherwise, include all nodes by default`)

	fs.StringSliceVar(&o.TenantList, "tenant-list", []string{}, `Comma-delimited list of tenant names to restart.
  E.g.:'--tenant-list=name1,name2,name3'`)

	fs.StringVar(&o.CustomSystemdUnitName, "systemd-unit", "", "Specify custom systemd unit name to restart")

	fs.StringVar(&rawSSHUnparsedArgs, "ssh-args", "",
		`This argument will be used when ssh-ing to the nodes. It may be used to override
the ssh command itself, ssh username or any additional arguments.
Double quotes are can be escaped with backward slash '\'.
Examples:
1) --ssh-args "pssh -A -J <some jump host> --yc-profile <YC profile name>"
2) --ssh-args "ssh -o ProxyCommand=\"...\""`)

	fs.StringSliceVar(&o.Datacenters, "dc", []string{},
		`Filter hosts by specific datacenter. The list is comma-delimited.
  E.g.: '--dc=ru-central1-a,ru-central1-b`)

	fs.StringSliceVar(&o.Hosts, "hosts", []string{},
		`Restart only specified hosts. You can specify a list of host FQDNs or a list of node ids,
but you can not mix host FQDNs and node ids in this option. The list is comma-delimited.
  E.g.: '--hosts=1,2,3' or '--hosts=fqdn1,fqdn2,fqdn3'`)

	fs.StringSliceVar(&o.ExcludeHosts, "exclude-hosts", []string{},
		`Comma-delimited list. Do not restart these hosts, even if they are explicitly specified in --hosts.`)

	fs.StringVar(&o.AvailabilityMode, "availability-mode", "strong",
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(options.AvailabilityModes, ", ")))

	fs.IntVar(&o.RestartDuration, "duration", DefaultRestartDurationSeconds,
		`CMS will release the node for maintenance for duration * restart-retry-number seconds. Any maintenance
after that would be considered a regular cluster failure`)

	fs.IntVar(&o.RestartRetryNumber, "restart-retry-number", DefaultRetryCount,
		fmt.Sprintf("How many times a node should be retried on error, default %v", DefaultRetryCount))

	fs.IntVar(&o.CMSQueryInterval, "cms-query-interval", DefaultCMSQueryIntervalSeconds,
		fmt.Sprintf("How often to query CMS while waiting for new permissions %v", DefaultCMSQueryIntervalSeconds))

	fs.StringVar(&startedUnparsedFlag, "started", "",
		fmt.Sprintf("Apply filter by node started time. Format: [<>%%Y-%%m-%%dT%%H:%%M:%%SZ], e.g. >2024-03-13T17:20:06Z"))

	fs.StringVar(&versionUnparsedFlag, "version", "",
		`Apply filter by node version.
Format: [(<|>|!=|~=)MAJOR.MINOR.PATCH|(==|!=)VERSION_STRING], e.g.:
'--version ~=24.1.2' or
'--version !=24.1.2-ydb-stable-hotfix-5'`)

	fs.BoolVar(&o.Continue, "continue", false,
		`Attempt to continue previous rolling restart, if there was one. The set of selected nodes
for this invocation must be the same as for the previous invocation, and this can not be verified at runtime since
the ydbops utility is stateless. Use at your own risk.`)

	fs.IntVar(&o.MaxStaticNodeId, "max-static-node-id", restarters.DefaultMaxStaticNodeId,
		`This argument is used to help ydbops distinguish storage and dynamic nodes.
Nodes with this nodeId or less will be considered storage.`)

	fs.BoolVar(&o.SuppressCompatibilityCheck, "suppress-compat-check", false,
		`By default, nodes within one cluster can differ by at most one major release. 
ydbops will try to figure out if you broke this rule by comparing before\after of some restarted node.`)

	profile.PopulateFromProfileLater(
		fs.StringVar, &o.KubeconfigPath, "kubeconfig",
		"",
		"[can specify in profile] Path to kubeconfig file.")

	profile.PopulateFromProfileLater(
		fs.StringVar, &o.K8sNamespace, "k8s-namespace",
		"",
		"[can specify in profile] Limit your operations to pods in this kubernetes namespace.")
}

func (o *RestartOptions) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}

func (o *RestartOptions) GetRestartDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.RestartDuration) * time.Duration(o.RestartRetryNumber))
}

func parseVersionFlag(versionUnparsedFlag string) (options.VersionSpec, error) {
	matches := majorMinorPatchRegexp.FindStringSubmatch(versionUnparsedFlag)
	if len(matches) == 5 {
		// `--version` value looks like (sign)major.minor.patch
		major, _ := strconv.Atoi(matches[2])
		minor, _ := strconv.Atoi(matches[3])
		patch, _ := strconv.Atoi(matches[4])
		return &options.MajorMinorPatchVersion{
			Sign:  matches[1],
			Major: major,
			Minor: minor,
			Patch: patch,
		}, nil
	}

	matches = rawRegexp.FindStringSubmatch(versionUnparsedFlag)
	if len(matches) == 3 {
		// `--version` value is an arbitrary string value, and will
		// be compared directly
		return &options.RawVersion{
			Sign: matches[1],
			Raw:  matches[2],
		}, nil
	}

	return nil, fmt.Errorf(
		"failed to interpret the value of `--version` flag. Read `ydbops restart --help` for more info on what is expected",
	)
}
