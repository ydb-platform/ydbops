package options

import (
	"fmt"
	"time"
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

func (v VersionSpec) String() string {
	return fmt.Sprintf("%s%v.%v.%v", v.Sign, v.Major, v.Minor, v.Patch)
}
