package options

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/profile"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	DefaultMaxStaticNodeID = 50000
)

var (
	startedUnparsedFlag string
	versionUnparsedFlag string
)

var (
	majorMinorPatchPattern = `^(>|<|!=|~=)(\d+|\*)\.(\d+|\*)\.(\d+|\*)$`
	majorMinorPatchRegexp  = regexp.MustCompile(majorMinorPatchPattern)

	rawPattern = `^(==|!=)(.*)$`
	rawRegexp  = regexp.MustCompile(rawPattern)
)

type TargetingOptions struct {
	AvailabilityMode string
	Datacenters      []string
	Hosts            []string
	ExcludeHosts     []string
	Version          string

	StartedTime *StartedTime
	VersionSpec VersionSpec

	Storage    bool
	Tenant     bool
	TenantList []string

	KubeconfigPath string
	K8sNamespace   string

	MaxStaticNodeID int
}

func (o *TargetingOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Storage, "storage", false, `Only include storage nodes. Otherwise, include all nodes by default`)

	fs.BoolVar(&o.Tenant, "tenant", false, `Only include tenant nodes. Otherwise, include all nodes by default`)

	fs.StringSliceVar(&o.TenantList, "tenant-list", []string{}, `Comma-delimited list of tenant names to restart.
  E.g.:'--tenant-list=name1,name2,name3'`)

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
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(AvailabilityModes, ", ")))

	fs.StringVar(&startedUnparsedFlag, "started", "",
		fmt.Sprintf(`Apply filter by node started time.
Format: "<>%%Y-%%m-%%dT%%H:%%M:%%SZ", quotes are necessary, otherwise shell treats '<' or '>' as stream redirection.
For example, --started ">2024-03-13T17:20:06Z" means all nodes started LATER than 2024 March 13, 17:20:06 UTC.
If you reverse the sign (--started ">2024-03-13T17:20:06Z"), you will select nodes with LARGER uptimes.`))

	fs.StringVar(&versionUnparsedFlag, "version", "",
		`Apply filter by node version.
Format: [(<|>|!=|~=)MAJOR.MINOR.PATCH|(==|!=)VERSION_STRING], e.g.:
'--version ~=24.1.2' or
'--version !=24.1.2-ydb-stable-hotfix-5'`)

	fs.IntVar(&o.MaxStaticNodeID, "max-static-node-id", DefaultMaxStaticNodeID,
		`This argument is used to help ydbops distinguish storage and dynamic nodes.
Nodes with this nodeId or less will be considered storage.`)

	profile.PopulateFromProfileLater(
		fs.StringVar, &o.KubeconfigPath, "kubeconfig",
		"",
		"[can specify in profile] Path to kubeconfig file.")

	profile.PopulateFromProfileLater(
		fs.StringVar, &o.K8sNamespace, "k8s-namespace",
		"",
		"[can specify in profile] Limit your operations to pods in this kubernetes namespace.")
}

func (o *TargetingOptions) Validate() error {
	if !collections.Contains(AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if len(o.KubeconfigPath) > 0 && len(o.K8sNamespace) == 0 {
		return fmt.Errorf("specified --kubeconfig, but not --k8s-namespace")
	}

	if o.MaxStaticNodeID < 0 {
		return fmt.Errorf("specified invalid max-static-node-id: %d. Must be positive", o.MaxStaticNodeID)
	}

	if len(o.TenantList) > 0 && !o.Tenant {
		return fmt.Errorf("--tenant-list specified, but --tenant is not explicitly specified." +
			"Please specify --tenant as well to clearly indicate your intentions")
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

		o.StartedTime = &StartedTime{
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

func parseVersionFlag(versionUnparsedFlag string) (VersionSpec, error) {
	matches := majorMinorPatchRegexp.FindStringSubmatch(versionUnparsedFlag)
	if len(matches) == 5 {
		// `--version` value looks like (sign)major.minor.patch
		major, _ := strconv.Atoi(matches[2])
		minor, _ := strconv.Atoi(matches[3])
		patch, _ := strconv.Atoi(matches[4])
		return &MajorMinorPatchVersion{
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
		return &RawVersion{
			Sign: matches[1],
			Raw:  matches[2],
		}, nil
	}

	return nil, fmt.Errorf(
		"failed to interpret the value of `--version` flag. Read `ydbops restart --help` for more info on what is expected",
	)
}

func (o *TargetingOptions) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}
