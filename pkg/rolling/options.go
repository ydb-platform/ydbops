package rolling

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
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
)

type RestartOptions struct {
	options.TargetingOptions

	RestartRetryNumber         int
	CMSQueryInterval           int
	NodesInflight              int
	DelayBetweenRestarts       time.Duration
	SuppressCompatibilityCheck bool

	RestartDuration int

	Continue bool

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

	fs.BoolVar(&o.Continue, "continue", false,
		`Attempt to continue previous rolling restart, if there was one. The set of selected nodes
for this invocation must be the same as for the previous invocation, and this can not be verified at runtime since
the ydbops utility is stateless. Use at your own risk.`)

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
}

func (o *RestartOptions) GetRestartDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.RestartDuration) * time.Duration(o.RestartRetryNumber))
}
