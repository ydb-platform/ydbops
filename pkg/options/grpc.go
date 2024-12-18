package options

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/profile"
)

const (
	GRPCDefaultTimeoutSeconds = 60
	GRPCDefaultPort           = 2135
)

type GRPC struct {
	Endpoint       string
	CaFile         string
	GRPCSecure     bool
	GRPCPort       int
	GRPCSkipVerify bool
	TimeoutSeconds int
}

func (o *GRPC) DefineFlags(fs *pflag.FlagSet) {
	profile.PopulateFromProfileLaterP(
		fs.StringVarP, &o.Endpoint, "endpoint", "e",
		"",
		fmt.Sprintf(`PROTOCOL://HOST[:PORT]
  A GRPC URL to connect to the YDB cluster. Default port is %v`, GRPCDefaultPort))

	fs.IntVar(&o.TimeoutSeconds, "grpc-timeout-seconds", GRPCDefaultTimeoutSeconds,
		"Wait this much before timing out any GRPC requests")

	fs.BoolVar(&o.GRPCSkipVerify, "grpc-skip-verify", false,
		"Do not verify server hostname when using grpcs")

	profile.PopulateFromProfileLater(
		fs.StringVar, &o.CaFile, "ca-file",
		"",
		"Path to root ca file, appends to system pool")
}

func (o *GRPC) Validate() error {
	if o.CaFile != "" {
		if !strings.Contains(o.Endpoint, "grpcs") {
			return fmt.Errorf("--ca-file must be specified only for secure connection")
		}

		if _, err := os.Stat(o.CaFile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("--ca-file file not found: %w", err)
		}
	}

	if o.TimeoutSeconds < 0 {
		return fmt.Errorf("invalid grpc timeout value specified: %d", o.TimeoutSeconds)
	}

	if !o.GRPCSecure && o.GRPCSkipVerify {
		return fmt.Errorf("unexpected --grpc-skip-verify with insecure grpc schema")
	}

	// skip validation if empty endpoint
	if o.Endpoint == "" {
		return nil
	}

	parsedURL, err := url.Parse(o.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse --endpoint: %w", err)
	}

	switch parsedURL.Scheme {
	case "grpcs":
		o.GRPCSecure = true
	case "grpc":
		o.GRPCSecure = false
	default:
		return fmt.Errorf("please specify the protocol in the endpoint explicitly: grpc or grpcs")
	}

	// Strip o.Endpoint from protocol and port number
	o.Endpoint = parsedURL.Hostname()

	switch parsedURL.Port() {
	case "":
		o.GRPCPort = GRPCDefaultPort
	default:
		port, _ := strconv.Atoi(parsedURL.Port())
		if port < 0 || port > 65536 {
			return fmt.Errorf("invalid port specified: %d, must be in range: (%d,%d)", port, 1, 65536)
		}
		o.GRPCPort = port
	}

	return nil
}
