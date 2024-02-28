package options

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

const (
	DefaultAuthType = "none"
)

type RootOptions struct {
	Auth     AuthOptions
	Endpoint string
	CaFile   string
}

var RootOptionsInstance = &RootOptions{}

func (o *RootOptions) Validate() error {
	if (o.Endpoint) == "" {
		return fmt.Errorf("specify a grpc endpoint with --endpoint")
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

	o.Auth.DefineFlags(fs)
}
