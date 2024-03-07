package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

const (
	CMSDefaultRetryWaitTime    = 60
	CMSDefaultAvailAbilityMode = "max"
	CMSDefaultAuthType         = "none"
	CMSDefaultTimeoutSeconds   = 60
)

type CMS struct {
	RetryWaitSeconds int
	TimeoutSeconds   int
}

func (cms *CMS) DefineFlags(fs *pflag.FlagSet) {
	fs.IntVarP(&cms.RetryWaitSeconds, "cms-wait-time-seconds", "", CMSDefaultRetryWaitTime,
		"CMS retry time in seconds")
	fs.IntVarP(&cms.TimeoutSeconds, "cms-timeout-seconds", "", CMSDefaultTimeoutSeconds,
		"CMS API response timeout in seconds")
}

func (cms *CMS) Validate() error {
	if cms.RetryWaitSeconds < 0 {
		return fmt.Errorf("invalid value specified: %d", cms.RetryWaitSeconds)
	}
	if cms.TimeoutSeconds < 0 {
		return fmt.Errorf("invalid value specified: %d", cms.TimeoutSeconds)
	}

	return nil
}
