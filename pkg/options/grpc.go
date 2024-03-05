package options

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

type GRPC struct {
	Endpoint   string
	CaFile     string
	GRPCSecure bool
	GRPCPort   int
}

func (o *GRPC) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Endpoint, "endpoint", "",
		"A GRPC address to connect to the YDB cluster")

	fs.StringVar(&o.CaFile, "ca-file", "", "Path to root ca file, overrides system pool")
}

func (o *GRPC) Validate() error {
	if o.CaFile != "" {
		if !strings.Contains(o.Endpoint, "grpcs") {
			return fmt.Errorf("root CA must be specified only for secure connection")
		}

		if _, err := os.Stat(o.CaFile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("root CA file not found: %v", err)
		}
	}

	if (o.Endpoint) == "" {
		return fmt.Errorf("specify a grpc endpoint with --endpoint")
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
	case "":
		// TODO should default GRPCSecure be true?
		o.GRPCSecure = true
	default:
    return fmt.Errorf("Please specify the protocol in the endpoint explicitly: grpc or grpcs. Currently specified: %s\n", parsedURL.Scheme)
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
