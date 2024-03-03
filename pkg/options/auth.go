package options

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"go.uber.org/zap"
)

const (
	DefaultCMSAuthEnvVar      = "YDB_TOKEN"
	DefaultCMSAuthIAMEndpoint = "iam.api.cloud.yandex.net"
)

var (
	Auths = map[string]Creds{
		"none": &CredsNone{},
		"env":  &CredsEnv{},
		"file": &CredsFile{},
		"iam":  &CredsIAM{},
	}
)

type (
	Creds interface {
		Options
		Token() (AuthToken, error)
	}
	AuthToken struct {
		Type   string
		Secret string
	}
	CredsNone struct{}
	CredsEnv  struct {
		Name string

		t AuthToken
	}
	CredsFile struct {
		Filename string

		t AuthToken
	}
	CredsIAM struct {
		KeyFilename string
		Endpoint    string
	}
)

type AuthOptions struct {
	Creds    Creds
	AuthType string
}

func (t AuthToken) Token() string {
	if t.Type == "" {
		return t.Secret
	}
	return fmt.Sprintf("%s %s", t.Type, t.Secret)
}

func (an *CredsNone) DefineFlags(_ *pflag.FlagSet) {}
func (an *CredsNone) Validate() error              { return nil }
func (an *CredsNone) Token() (AuthToken, error)    { return AuthToken{}, nil }

func (ae *CredsEnv) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&ae.Name, "auth-env-name", "", DefaultCMSAuthEnvVar,
		"Authentication environment variable name (type: env)")
}

func (ae *CredsEnv) Validate() error {
	if len(ae.Name) == 0 {
		return fmt.Errorf("auth env variable name empty")
	}
	return nil
}

func (ae *CredsEnv) Token() (AuthToken, error) {
	if ae.t.Secret != "" {
		return ae.t, nil
	}

	zap.S().Debugf("Read auth token from %s variable", ae.Name)
	val, ok := os.LookupEnv(ae.Name)
	if !ok {
		return AuthToken{}, fmt.Errorf("failed to lookup variable: %s", ae.Name)
	}
	return AuthToken{
		Type:   "OAuth",
		Secret: val,
	}, nil
}

func (af *CredsFile) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&af.Filename, "auth-file-token", "", "",
		"Authentication file token name (type: file)")
}

func (af *CredsFile) Validate() error {
	if len(af.Filename) != 0 {
		if _, err := os.Stat(af.Filename); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("auth password file not exists: %v", err)
		}
	}
	return nil
}

func (af *CredsFile) Token() (AuthToken, error) {
	if af.t.Secret != "" {
		return af.t, nil
	}

	zap.S().Debugf("Read auth token from %s file", af.Filename)
	b, err := os.ReadFile(af.Filename)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to read token file: %v", err)
	}
	return AuthToken{
		Type:   "OAuth",
		Secret: string(b),
	}, nil
}

func (at *CredsIAM) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&at.KeyFilename, "auth-iam-key-file", "", "",
		"Authentication iam key file path (type: iam)")
	fs.StringVarP(&at.Endpoint, "auth-iam-endpoint", "", DefaultCMSAuthIAMEndpoint,
		"Authentication iam endpoint (type: iam)")
}

func (at *CredsIAM) Validate() error {
	if len(at.KeyFilename) != 0 {
		if _, err := os.Stat(at.KeyFilename); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("auth iam key file %s not exists: %v", at.KeyFilename, err)
		}
	}
	if len(at.Endpoint) == 0 {
		return fmt.Errorf("empty iam endpoint specified")
	}
	return nil
}

func (at *CredsIAM) Token() (AuthToken, error) {
	// TODO support IAM authorizaion
	return AuthToken{}, nil
}

func (o *AuthOptions) Validate() error {
	if !util.Contains(util.Keys(Auths), o.AuthType) {
		return fmt.Errorf("invalid auth type specified: %s, use one of: %+v", o.AuthType, util.Keys(Auths))
	}

	o.Creds = Auths[o.AuthType]
	if err := o.Creds.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *AuthOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.AuthType, "auth-type", "", DefaultAuthType,
		fmt.Sprintf("Authentication types: %+v", util.Keys(Auths)))

	for _, auth := range Auths {
		auth.DefineFlags(fs)
	}
}
