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

const (
	DefaultAuthType = "none"
	GRPCDefaultPort = 2135
)

type RootOptions struct {
	Auth       AuthOptions
	Endpoint   string
	CaFile     string
	Verbose    bool
	GRPCSecure bool
	GRPCPort   int
}

var RootOptionsInstance = &RootOptions{
}

func (o *RootOptions) Validate() error {
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
		return fmt.Errorf("found the schema to not be grpc or grpcs: %s\n", parsedURL.Scheme)
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

	if o.CaFile != "" {
		if !strings.Contains(o.Endpoint, "grpcs") {
			return fmt.Errorf("root CA must be specified only for secure connection")
		}

		if _, err := os.Stat(o.CaFile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("root CA file not found: %v", err)
		}
	}

  o.Auth.Validate()

	return nil
}

func (o *RootOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Endpoint, "endpoint", "",
		"TODO GRPC addresses which will be used to connect to cluster")

	fs.StringVar(&o.CaFile, "ca-file", "", "TODO path to root ca file")

	fs.BoolVar(&o.Verbose, "verbose", false, "TODO should enable verbose output")

	o.Auth.DefineFlags(fs)
}
