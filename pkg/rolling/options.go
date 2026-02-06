package rolling

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/internal/collections"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	DefaultRetryCount              = 3
	DefaultCMSQueryIntervalSeconds = 10
	DefaultRestartDurationSeconds  = 60
	DefaultNodesInflight           = 1
	DefaultDelayBetweenRestarts    = time.Second
	DefaultOrderingKey             = ClusterOrderingKey
	DefaultTenantsInflight         = 1
)

type RestartOptions struct {
	options.TargetingOptions

	RestartRetryNumber         int
	CMSQueryInterval           int
	NodesInflight              int
	DelayBetweenRestarts       time.Duration
	SuppressCompatibilityCheck bool
	CleanupOnExit              bool

	OrderingKey     string
	TenantsInflight int

	RestartDuration int

	SSHArgs []string

	CustomSystemdUnitName string
}

var rawSSHUnparsedArgs string

func (o *RestartOptions) Validate() error {
	err := o.TargetingOptions.Validate()
	if err != nil {
		return err
	}

	if o.CMSQueryInterval < 0 {
		return fmt.Errorf("specified invalid cms query interval seconds: %d. Must be positive", o.CMSQueryInterval)
	}

	if o.RestartRetryNumber < 0 {
		return fmt.Errorf("specified invalid restart retry number: %d. Must be positive", o.RestartRetryNumber)
	}

	if o.RestartDuration < 0 {
		return fmt.Errorf("specified invalid restart duration: %d. Must be positive", o.RestartDuration)
	}

	if !collections.Contains(OrderingKeyChoices, o.OrderingKey) {
		return fmt.Errorf("specified invalid ordering key: %s, valid values are %s",
			o.OrderingKey,
			strings.Join(OrderingKeyChoices, ","))
	}

	if o.TenantsInflight < 1 {
		return fmt.Errorf("specified invalid inflight tenants: %d. Must be greater than or equal to 1", o.TenantsInflight)
	}

	o.SSHArgs = utils.ParseSSHArgs(rawSSHUnparsedArgs)

	return nil
}

func (o *RestartOptions) DefineFlags(fs *pflag.FlagSet) {
	o.TargetingOptions.DefineFlags(fs)

	fs.StringVar(&o.CustomSystemdUnitName, "systemd-unit", "", "Specify custom systemd unit name to restart")

	fs.StringVar(&rawSSHUnparsedArgs, "ssh-args", "",
		`This argument will be used when ssh-ing to the nodes. It may be used to override
the ssh command itself, ssh username or any additional arguments.
Double quotes are can be escaped with backward slash '\'.
Examples:
1) --ssh-args "pssh -A -J <some jump host> --yc-profile <YC profile name>"
2) --ssh-args "ssh -o ProxyCommand=\"...\""`)

	fs.IntVar(&o.RestartRetryNumber, "restart-retry-number", DefaultRetryCount,
		fmt.Sprintf("How many times a node should be retried on error, default %v", DefaultRetryCount))

	fs.IntVar(&o.CMSQueryInterval, "cms-query-interval", DefaultCMSQueryIntervalSeconds,
		fmt.Sprintf("How often to query CMS while waiting for new permissions %v", DefaultCMSQueryIntervalSeconds))

	fs.IntVar(&o.RestartDuration, "duration", DefaultRestartDurationSeconds,
		`CMS will release the node for maintenance for duration * restart-retry-number seconds. Any maintenance
after that would be considered a regular cluster failure`)

	fs.BoolVar(&o.SuppressCompatibilityCheck, "suppress-compat-check", false,
		`By default, nodes within one cluster can differ by at most one major release.
ydbops will try to figure out if you broke this rule by comparing before\after of some restarted node.`)

	fs.IntVar(&o.NodesInflight, "nodes-inflight", DefaultNodesInflight,
		`The limit on the number of simultaneous node restarts`)

	fs.DurationVar(&o.DelayBetweenRestarts, "delay-between-restarts", DefaultDelayBetweenRestarts,
		`Delay between two consecutive restarts. E.g. '60s', '2m'. The number of simultaneous restarts is limited by 'nodes-inflight'.`)

	fs.BoolVar(&o.CleanupOnExit, "cleanup-on-exit", true,
		`When enabled, attempt to drop the maintenance task if the utility is killed by SIGTERM.`)

	fs.StringVar(&o.OrderingKey, "ordering-key", DefaultOrderingKey,
		fmt.Sprintf("Re-orders nodes for restart by a key. Available choices: %s", strings.Join(OrderingKeyChoices, ", ")))

	fs.IntVar(&o.TenantsInflight, "tenants-inflight", DefaultTenantsInflight,
		`Maximum number of distinct tenants to restart in parallel. Example: 2 means only up to 2 tenants can have nodes restarting at the same time.`)
}

func (o *RestartOptions) GetRestartDuration(nNodes int) *durationpb.Duration {
	singleBatchRestartTime := time.Second * time.Duration(o.RestartDuration) * time.Duration(o.RestartRetryNumber)
	singleBatchWithWait := singleBatchRestartTime + o.DelayBetweenRestarts
	maximumTotalBatches := int(math.Ceil(float64(nNodes) / float64(o.NodesInflight)))

	finalDuration := time.Duration(maximumTotalBatches) * singleBatchWithWait
	return durationpb.New(finalDuration)
}
