package options

import "time"

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
