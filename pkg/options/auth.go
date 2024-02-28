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
	Auths = map[string]Auth{
		"none": &AuthNone{},
		"env":  &AuthEnv{},
		"file": &AuthFile{},
		"iam":  &AuthIAM{},
	}
)

type (
	Auth interface {
		Options
		Token() (AuthToken, error)
	}
	AuthToken struct {
		Type   string
		Secret string
	}
	AuthNone struct{}
	AuthEnv  struct {
		Name string

		t AuthToken
	}
	AuthFile struct {
		Filename string

		t AuthToken
	}
	AuthIAM struct {
		KeyFilename string
		Endpoint    string
	}
)

type AuthOptions struct {
	Auth     Auth
	AuthType string
}

func (t AuthToken) Token() string {
	if t.Type == "" {
		return t.Secret
	}
	return fmt.Sprintf("%s %s", t.Type, t.Secret)
}

func (an *AuthNone) DefineFlags(_ *pflag.FlagSet) {}
func (an *AuthNone) Validate() error              { return nil }
func (an *AuthNone) Token() (AuthToken, error)    { return AuthToken{}, nil }

func (ae *AuthEnv) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&ae.Name, "cms-auth-env-name", "", DefaultCMSAuthEnvVar,
		"CMS Authentication environment variable name (type: env)")
}

func (ae *AuthEnv) Validate() error {
	if len(ae.Name) == 0 {
		return fmt.Errorf("auth env variable name empty")
	}
	return nil
}

func (ae *AuthEnv) Token() (AuthToken, error) {
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

func (af *AuthFile) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&af.Filename, "cms-auth-file-token", "", "",
		"CMS Authentication file token name (type: file)")
}

func (af *AuthFile) Validate() error {
	if len(af.Filename) != 0 {
		if _, err := os.Stat(af.Filename); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("auth password file not exists: %v", err)
		}
	}
	return nil
}

func (af *AuthFile) Token() (AuthToken, error) {
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

func (at *AuthIAM) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&at.KeyFilename, "cms-auth-iam-key-file", "", "",
		"CMS Authentication iam key file path (type: iam)")
	fs.StringVarP(&at.Endpoint, "cms-auth-iam-endpoint", "", DefaultCMSAuthIAMEndpoint,
		"CMS Authentication iam endpoint (type: iam)")
}

func (at *AuthIAM) Validate() error {
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

func (at *AuthIAM) Token() (AuthToken, error) {
	return AuthToken{}, nil
}

func (o *AuthOptions) Validate() error {
	if !util.Contains(util.Keys(Auths), o.AuthType) {
		return fmt.Errorf("invalid auth type specified: %s, use one of: %+v", o.AuthType, util.Keys(Auths))
	}

	o.Auth = Auths[o.AuthType]
	if err := o.Auth.Validate(); err != nil {
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
