package options

import (
	"fmt"
	"net/url"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/profile"
)

const JWTTokenType = "urn:ietf:params:oauth:token-type:jwt"

type AuthOAuth2TokenExchange struct {
	TokenEndpoint    string
	SubjectToken     string
	SubjectTokenFile string
	SubjectTokenType string
	ActorTokenFile   string
	ActorTokenType   string
}

func (a *AuthOAuth2TokenExchange) DefineFlags(fs *pflag.FlagSet) {
	profile.PopulateFromProfileLater(fs.StringVar, &a.TokenEndpoint, "oauth2-token-endpoint", "",
		"HTTPS OAuth2 token exchange URL (RFC 8693).")
	profile.PopulateFromProfileLater(fs.StringVar, &a.SubjectTokenFile, "oauth2-subject-token-file", "",
		"Subject token file; reread when exchanging credentials, including during refresh.")
	profile.PopulateFromProfileLater(fs.StringVar, &a.SubjectToken, "oauth2-subject-token", "",
		"Literal subject token or target identity; mutually exclusive with --oauth2-subject-token-file.")
	profile.PopulateFromProfileLater(fs.StringVar, &a.SubjectTokenType, "oauth2-subject-token-type", "",
		"Subject token type; defaults to the JWT type for a file, required for a literal subject.")
	profile.PopulateFromProfileLater(fs.StringVar, &a.ActorTokenFile, "oauth2-actor-token-file", "",
		"Optional actor token file; reread when exchanging credentials, including during refresh.")
	profile.PopulateFromProfileLater(fs.StringVar, &a.ActorTokenType, "oauth2-actor-token-type", "",
		"Actor token type; defaults to the JWT type when an actor file is configured.")
}

func (a *AuthOAuth2TokenExchange) Validate() error {
	u, err := url.Parse(a.TokenEndpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("--oauth2-token-endpoint must be an HTTPS URL without credentials or a fragment")
	}
	if a.SubjectToken == "" && a.SubjectTokenFile == "" {
		return fmt.Errorf("specify --oauth2-subject-token or --oauth2-subject-token-file")
	}
	if a.SubjectToken != "" && a.SubjectTokenFile != "" {
		return fmt.Errorf("--oauth2-subject-token and --oauth2-subject-token-file are mutually exclusive")
	}
	if a.SubjectToken != "" && a.SubjectTokenType == "" {
		return fmt.Errorf("--oauth2-subject-token-type is required for a literal subject")
	}
	if a.ActorTokenFile == "" && a.ActorTokenType != "" {
		return fmt.Errorf("--oauth2-actor-token-type requires --oauth2-actor-token-file")
	}
	if a.SubjectTokenType == "" {
		a.SubjectTokenType = JWTTokenType
	}
	if a.ActorTokenFile != "" && a.ActorTokenType == "" {
		a.ActorTokenType = JWTTokenType
	}
	return nil
}
