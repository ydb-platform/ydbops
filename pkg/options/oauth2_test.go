package options

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/profile"
)

func TestOAuth2Validation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config AuthOAuth2TokenExchange
		err    string
	}{
		{"missing endpoint", AuthOAuth2TokenExchange{}, "HTTPS"},
		{"insecure endpoint", AuthOAuth2TokenExchange{TokenEndpoint: "http://sts.example/token"}, "HTTPS"},
		{"missing host", AuthOAuth2TokenExchange{TokenEndpoint: "https:///token"}, "HTTPS"},
		{"URL credentials", AuthOAuth2TokenExchange{TokenEndpoint: "https://user:password@sts.example/token"}, "HTTPS"},
		{"URL fragment", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token#fragment"}, "HTTPS"},
		{"missing subject", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token"}, "specify"},
		{"two subjects", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectToken: "id", SubjectTokenFile: "token"}, "mutually exclusive"},
		{"literal without type", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectToken: "jwt"}, "required"},
		{"literal JWT with type", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectToken: "jwt", SubjectTokenType: JWTTokenType}, ""},
		{"actor type without file", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectTokenFile: "token", ActorTokenType: JWTTokenType}, "requires"},
		{"file", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectTokenFile: "token"}, ""},
		{"literal and actor", AuthOAuth2TokenExchange{TokenEndpoint: "https://sts.example/token", SubjectToken: "id", SubjectTokenType: "urn:example:identity", ActorTokenFile: "token"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			expectedType := tc.config.SubjectTokenType
			if expectedType == "" {
				expectedType = JWTTokenType
			}
			err := tc.config.Validate()
			if tc.err != "" {
				g.Expect(err).To(MatchError(ContainSubstring(tc.err)))
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc.config.SubjectTokenType).To(Equal(expectedType))
			if tc.config.ActorTokenFile != "" {
				g.Expect(tc.config.ActorTokenType).To(Equal(JWTTokenType))
			}
		})
	}
}

func authFlags(t *testing.T) (*AuthOptions, *pflag.FlagSet) {
	t.Helper()
	previous := Auths
	Auths = map[AuthType]Options{
		None: &AuthNone{}, Static: &AuthStatic{}, IamToken: &AuthIAMToken{},
		IamCreds: &AuthIAMCreds{}, IamMetadata: &AuthIAMMetadataCredentials{},
		OAuth2TokenExchange: &AuthOAuth2TokenExchange{},
	}
	t.Cleanup(func() { Auths = previous })
	opts := &AuthOptions{}
	fs := pflag.NewFlagSet(t.Name(), pflag.ContinueOnError)
	opts.DefineFlags(fs)
	return opts, fs
}

func TestOAuth2Selection(t *testing.T) {
	for _, name := range []string{"token-endpoint", "subject-token", "subject-token-file", "subject-token-type", "actor-token-file", "actor-token-type"} {
		t.Run("partial "+name, func(t *testing.T) {
			g := NewWithT(t)
			opts, fs := authFlags(t)
			t.Setenv(DefaultAuthEnvVar, "implicit-token")
			g.Expect(fs.Parse([]string{"--oauth2-" + name, "value"})).To(Succeed())
			g.Expect(opts.Validate()).NotTo(Succeed())
			g.Expect(opts.Type).To(Equal(OAuth2TokenExchange))
		})
	}
	t.Run("explicit exchange overrides implicit token", func(t *testing.T) {
		g := NewWithT(t)
		opts, fs := authFlags(t)
		t.Setenv(DefaultAuthEnvVar, "implicit-token")
		g.Expect(fs.Parse([]string{"--oauth2-token-endpoint", "https://sts.example/token", "--oauth2-subject-token-file", "token"})).To(Succeed())
		g.Expect(opts.Validate()).To(Succeed())
		g.Expect(opts.Type).To(Equal(OAuth2TokenExchange))
		g.Expect(fs.Parse([]string{"--token-file", "another-token"})).To(Succeed())
		g.Expect(opts.Validate()).To(MatchError(ContainSubstring("more than one")))
	})
	t.Run("no exchange settings preserves existing selection", func(t *testing.T) {
		g := NewWithT(t)
		opts, _ := authFlags(t)
		t.Setenv(DefaultAuthEnvVar, "implicit-token")
		g.Expect(opts.Validate()).To(Succeed())
		g.Expect(opts.Type).To(Equal(IamToken))
	})
}

func TestOAuth2Profile(t *testing.T) {
	g := NewWithT(t)
	opts, fs := authFlags(t)
	path := filepath.Join(t.TempDir(), "profile.yaml")
	g.Expect(os.WriteFile(path, []byte(`current-profile: workload
profiles:
  workload:
    oauth2-token-endpoint: https://sts.example/token
    oauth2-subject-token: target-identity
    oauth2-subject-token-type: urn:example:identity
    oauth2-actor-token-file: /var/run/projected/token
    oauth2-actor-token-type: urn:example:assertion
`), 0600)).To(Succeed())
	g.Expect(fs.Parse([]string{"--oauth2-token-endpoint", "https://override.example/token"})).To(Succeed())
	g.Expect(profile.FillDefaultsFromActiveProfile(path, "")).To(Succeed())
	g.Expect(opts.Validate()).To(Succeed())
	g.Expect(opts.Creds).To(Equal(&AuthOAuth2TokenExchange{
		TokenEndpoint: "https://override.example/token", SubjectToken: "target-identity",
		SubjectTokenType: "urn:example:identity", ActorTokenFile: "/var/run/projected/token", ActorTokenType: "urn:example:assertion",
	}))
}
