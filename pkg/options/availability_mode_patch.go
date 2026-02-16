package options

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
)

// AvailabilityModeSmartValue is the proto enum value for AVAILABILITY_MODE_SMART.
// This patches the proto enum maps until ydb-go-genproto is updated with the new value.
const AvailabilityModeSmartValue = Ydb_Maintenance.AvailabilityMode(4)

func init() {
	Ydb_Maintenance.AvailabilityMode_name[4] = "AVAILABILITY_MODE_SMART"
	Ydb_Maintenance.AvailabilityMode_value["AVAILABILITY_MODE_SMART"] = 4
}
