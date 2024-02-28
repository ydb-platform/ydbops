package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

const (
	GRPCDefaultPort           = 2135
	GRPCDefaultTimeoutSeconds = 60
)

type GRPC struct {
	Port           int
	Secure         bool
	TimeoutSeconds int
}

func (grpc *GRPC) DefineFlags(fs *pflag.FlagSet) {
	fs.IntVarP(&grpc.Port, "grpc-port", "", GRPCDefaultPort,
		"GRPC port available on all addresses")
	fs.BoolVarP(&grpc.Secure, "grpc-secure", "", grpc.Secure,
		"GRPC or GRPCS protocol to use")
	fs.IntVarP(&grpc.TimeoutSeconds, "grpc-api-timeout-seconds", "", GRPCDefaultTimeoutSeconds,
		"CMS API response timeout in seconds")
}

func (grpc *GRPC) Validate() error {
	if grpc.Port < 0 || grpc.Port > 65536 {
		return fmt.Errorf("invalid port specified: %d, must be in range: (%d,%d)", grpc.Port, 1, 65536)
	}
	if grpc.TimeoutSeconds < 0 {
		return fmt.Errorf("invalid value specified: %d", grpc.TimeoutSeconds)
	}

	return nil
}
