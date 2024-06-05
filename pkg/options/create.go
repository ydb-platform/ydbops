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
	HostFQDN            string
	MaintenanceDuration int
	AvailabilityMode    string
}

const (
	DefaultMaintenanceDurationSeconds = 3600
)

func (o *MaintenanceCreateOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.HostFQDN, "host-fqdn", "",
		"Request the host with this FQDN from the cluster")

	fs.StringVar(&o.AvailabilityMode, "availability-mode", "strong",
		fmt.Sprintf("Availability mode. Available choices: %s", strings.Join(AvailabilityModes, ", ")))

	fs.IntVar(&o.MaintenanceDuration, "duration", DefaultMaintenanceDurationSeconds,
		fmt.Sprintf("The time you need to perform host maintenance, in seconds. Default: %v",
			DefaultMaintenanceDurationSeconds))
}

func (o *MaintenanceCreateOpts) GetMaintenanceDuration() *durationpb.Duration {
	return durationpb.New(time.Second * time.Duration(o.MaintenanceDuration))
}

func (o *MaintenanceCreateOpts) Validate() error {
	if !collections.Contains(AvailabilityModes, o.AvailabilityMode) {
		return fmt.Errorf("specified a non-existing availability mode: %s", o.AvailabilityMode)
	}

	if o.MaintenanceDuration < 0 {
		return fmt.Errorf("specified invalid maintenance duration seconds: %d. Must be positive", o.MaintenanceDuration)
	}
	return nil
}

func (o *MaintenanceCreateOpts) GetAvailabilityMode() Ydb_Maintenance.AvailabilityMode {
	title := strings.ToUpper(fmt.Sprintf("availability_mode_%s", o.AvailabilityMode))
	value := Ydb_Maintenance.AvailabilityMode_value[title]

	return Ydb_Maintenance.AvailabilityMode(value)
}
