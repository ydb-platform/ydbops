package options

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/internal/collections"
)

type MaintenanceCreateOpts struct {
	HostFQDNs                  []string
	MaintenanceDurationSeconds int
	AvailabilityMode           string
}

const (
	DefaultMaintenanceDurationSeconds = 3600
)

func (o *MaintenanceCreateOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.HostFQDNs, "hosts", []string{},
		"Request the hosts with these FQDNs from the cluster")

	fs.StringVar(&o.AvailabilityMode, "availability-mode", "strong",
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(AvailabilityModes, ", ")))

	fs.IntVar(&o.MaintenanceDurationSeconds, "duration", DefaultMaintenanceDurationSeconds,
		fmt.Sprintf("How much time do you need for maintenance, in seconds. Default: %v",
			DefaultMaintenanceDurationSeconds))
}

func (o *MaintenanceCreateOpts) GetMaintenanceDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.MaintenanceDurationSeconds))
}

func (o *MaintenanceCreateOpts) Validate() error {
	if len(o.HostFQDNs) == 0 {
		return fmt.Errorf("--hosts unspecified")
	}

	if !collections.Contains(AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if o.MaintenanceDurationSeconds <= 0 {
		return fmt.Errorf("specified invalid maintenance duration seconds: %d. Must be positive", o.MaintenanceDurationSeconds)
	}
	return nil
}

// TODO this is copy paste? move to internals
func (o *MaintenanceCreateOpts) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}

type TaskIdOpts struct {
	TaskID string
}

func (o *TaskIdOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TaskID, "task-id", "",
		"ID of your maintenance task (result of `ydbops maintenance host`)")
}

func (o *TaskIdOpts) Validate() error {
	if o.TaskID == "" {
		return fmt.Errorf("--task-id unspecified, argument required")
	}
	return nil
}

type CompleteOpts struct {
	HostFQDNs []string
}

func (o *CompleteOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.HostFQDNs, "hosts", []string{},
		"FQDNs of hosts with completed maintenance")
}

func (o *CompleteOpts) Validate() error {
	if len(o.HostFQDNs) == 0 {
		return fmt.Errorf("--hosts unspecified")
	}
	return nil
}
