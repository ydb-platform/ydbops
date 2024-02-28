package options

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydb-ops/internal/util"
)

var (
	CMSAvailabilityModes = []string{"max", "keep", "force"}
)

const (
	CMSDefaultRetryWaitTime    = 60
	CMSDefaultAvailAbilityMode = "max"
	CMSDefaultAuthType         = "none"
)

type CMS struct {
	AvailabilityMode string
	RetryWaitSeconds int
}

func (cms *CMS) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&cms.AvailabilityMode, "cms-availability-mode", "", CMSDefaultAvailAbilityMode,
		fmt.Sprintf("CMS Availability mode (%+v)", CMSAvailabilityModes))
	fs.IntVarP(&cms.RetryWaitSeconds, "cms-wait-time-seconds", "", CMSDefaultRetryWaitTime,
		"CMS retry time in seconds")
}

func (cms *CMS) Validate() error {
	if !util.Contains(CMSAvailabilityModes, cms.AvailabilityMode) {
		return fmt.Errorf("invalid availability mode specified: %v, use one of: %+v", cms.AvailabilityMode, CMSAvailabilityModes)
	}
	if cms.RetryWaitSeconds < 0 {
		return fmt.Errorf("invalid value specified: %d", cms.RetryWaitSeconds)
	}

	return nil
}
