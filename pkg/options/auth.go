package options

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

const (
	DefaultAuthEnvVar = "YDB_TOKEN"

	DefaultStaticPasswordEnvVar = "YDB_PASSWORD"
	DefaultStaticUserEnvVar     = "YDB_USER"

	DefaultServiceAccountKeyFile = "SA_KEY_FILE"

	DefaultAuthIAMEndpoint = "iam.api.cloud.yandex.net"
)

type AuthType string

const (
	Unset          AuthType = "unset"
	None           AuthType = "none"
	Static         AuthType = "static"
	IamToken       AuthType = "iam-token"
	IamCreds       AuthType = "iam-creds"
	MultipleAtOnce AuthType = "multiple-at-once"
)

var (
	Auths = map[AuthType]Options{
		None:     &AuthNone{},
		Static:   &AuthStatic{},
		IamToken: &AuthIAMToken{},
		IamCreds: &AuthIAMCreds{},
		// TODO support OAuth
	}
)

type (
	AuthNone struct{}

	AuthStatic struct {
		User         string
		PasswordFile string
		Password     string
		NoPassword   bool
	}

	AuthIAMToken struct {
		TokenFile string
		Token     string
	}

	AuthIAMCreds struct {
		KeyFilename string
		Endpoint    string
	}
)

type AuthOptions struct {
	Creds Options
	Type  AuthType
}

func (an *AuthNone) DefineFlags(_ *pflag.FlagSet) {}

func (an *AuthNone) Validate() error { return nil }

func (a *AuthStatic) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.User, "user", "",
		fmt.Sprintf(`User name to authenticate with
User name search order:
	1. This option
	2. "%s" environment variable`, DefaultStaticUserEnvVar))

	fs.StringVar(&a.PasswordFile, "password-file", "",
		fmt.Sprintf(`File with password to authenticate with.
Password search order:
	1. This option
	2. "%s" environment variable (interpreted as password, not as a filepath)`, DefaultStaticPasswordEnvVar))
	fs.BoolVar(&a.NoPassword, "no-password", false,
		"TODO NOT IMPLEMENTED Do not ask for user password (if empty) (default: false)")
}

func (a *AuthStatic) Validate() error {
	if a.PasswordFile != "" {
		content, err := os.ReadFile(a.PasswordFile)
		if err != nil {
			return fmt.Errorf("Error reading file with static password: %w", err)
		}
		a.Password = string(content)
		return nil
	}

	if value, present := os.LookupEnv(DefaultStaticPasswordEnvVar); present {
		a.Password = value
		return nil
	}

	return fmt.Errorf(
		"Failed to discover the password: neither --password nor environment variable \"%s\" seem to be defined.",
		DefaultStaticPasswordEnvVar,
	)
}

func (a *AuthIAMToken) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.TokenFile, "token-file", "",
		`IAM token file
Token search order:
	1. This option
	2. "YDB_TOKEN" environment variable (interpreted as token, not as a filepath)`)
}

func (a *AuthIAMToken) Validate() error {
	if len(a.TokenFile) == 0 {
		if envToken, present := os.LookupEnv(DefaultAuthEnvVar); present {
			a.Token = envToken
			return nil
		} else {
			return fmt.Errorf(
				"failed to discover the token: neither --token-file nor environment variable \"%s\" seem to be defined.",
				DefaultAuthEnvVar,
			)
		}
	}

	content, err := os.ReadFile(a.TokenFile)
	if err != nil {
		return fmt.Errorf("failed to read the file specified in --token-file: %w", err)
	}

	a.Token = string(content)

	return nil
}

func (a *AuthIAMCreds) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&a.KeyFilename, "sa-key-file", "", "",
		fmt.Sprintf(`Service account key file
Definition priority:
	1. This option
	2. "%s" environment variable (interpreted as path to the file)`, DefaultServiceAccountKeyFile))
	fs.StringVarP(&a.Endpoint, "iam-endpoint", "", DefaultAuthIAMEndpoint,
		"Authentication iam endpoint")
}

func (a *AuthIAMCreds) Validate() error {
	if len(a.KeyFilename) != 0 {
		if _, err := os.Stat(a.KeyFilename); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("auth iam key file %s not exists: %v", a.KeyFilename, err)
		}
	}

	if envKeyFilename, present := os.LookupEnv(DefaultServiceAccountKeyFile); present {
		a.KeyFilename = envKeyFilename
	}

	if len(a.KeyFilename) == 0 {
		return fmt.Errorf("empty service account key filename specified")
	}

	if len(a.Endpoint) == 0 {
		return fmt.Errorf("empty iam endpoint specified")
	}
	return nil
}

func determineExplicitAuthType() AuthType {
	authType := map[AuthType]bool{}

	if static, ok := Auths[Static]; ok && static.(*AuthStatic).User != "" {
		_, passwordVarPresent := os.LookupEnv(DefaultStaticPasswordEnvVar)
		if static.(*AuthStatic).PasswordFile != "" || passwordVarPresent {
			authType[Static] = true
		}
	}

	if static, ok := Auths[IamToken]; ok && static.(*AuthIAMToken).TokenFile != "" {
		authType[IamToken] = true
	}

	if static, ok := Auths[IamCreds]; ok && static.(*AuthIAMCreds).Endpoint != "" && static.(*AuthIAMCreds).KeyFilename != "" {
		authType[IamCreds] = true
	}

	result := Unset
	for k := range authType {
		if authType[k] {
			if result == Unset {
				result = k
				continue
			}

			return MultipleAtOnce
		}
	}

	return result
}

func determineImplicitAuthType() AuthType {
	if _, present := os.LookupEnv(DefaultAuthEnvVar); present {
		return IamToken
	}

	_, userPresent := os.LookupEnv(DefaultStaticUserEnvVar)
	_, passwordPresent := os.LookupEnv(DefaultStaticPasswordEnvVar)

	if userPresent && passwordPresent {
		return Static
	}

	return None
}

func (o *AuthOptions) Validate() error {
	explicitlyActiveAuthType := determineExplicitAuthType()
	if explicitlyActiveAuthType == MultipleAtOnce {
		return fmt.Errorf("please specify exactly one authorization option. You specified more than one.")
	}

	activeAuthType := explicitlyActiveAuthType
	if activeAuthType == Unset {
		activeAuthType = determineImplicitAuthType()
	}

	o.Type = activeAuthType
	o.Creds = Auths[activeAuthType]
	Logger.Debugf("Determined auth type: %s", activeAuthType)

	if err := o.Creds.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *AuthOptions) DefineFlags(fs *pflag.FlagSet) {
	for _, auth := range Auths {
		auth.DefineFlags(fs)
	}
}
