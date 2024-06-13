package create

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/options"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Options struct {
	HostFQDNs                  []string
	MaintenanceDurationSeconds int
	AvailabilityMode           string
}

func (o *Options) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}

func (o *Options) GetMaintenanceDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.MaintenanceDurationSeconds))
}

const (
	DefaultMaintenanceDurationSeconds = 3600
)

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	// TODO(shmel1k@): move to 'WithAvailbilityModes' helper.
	fs.StringSliceVar(&o.HostFQDNs, "hosts", []string{},
		"Request the hosts with these FQDNs from the cluster")

	fs.StringVar(&o.AvailabilityMode, "availability-mode", "strong",
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(options.AvailabilityModes, ", ")))

	fs.IntVar(&o.MaintenanceDurationSeconds, "duration", DefaultMaintenanceDurationSeconds,
		fmt.Sprintf("How much time do you need for maintenance, in seconds. Default: %v",
			DefaultMaintenanceDurationSeconds))
}

func (o *Options) Validate() error {
	if len(o.HostFQDNs) == 0 {
		return fmt.Errorf("--hosts unspecified")
	}

	if !collections.Contains(options.AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if o.MaintenanceDurationSeconds <= 0 {
		return fmt.Errorf("specified invalid maintenance duration seconds: %d. Must be positive", o.MaintenanceDurationSeconds)
	}
	return nil
}
