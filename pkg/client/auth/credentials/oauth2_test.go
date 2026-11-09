package credentials

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/onsi/ginkgo/v2" // Register the Ginkgo flags passed to every test package by CI.
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func tokenServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	// The pinned SDK uses http.DefaultTransport; trust only this test server during the test.
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() {
		server.Close()
		http.DefaultTransport = previous
	})
	return server
}

func TestOAuth2ExchangeAndCache(t *testing.T) {
	for _, source := range []string{"file", "literal", "actor"} {
		t.Run(source, func(t *testing.T) {
			g := NewWithT(t)
			path := filepath.Join(t.TempDir(), "token")
			g.Expect(os.WriteFile(path, []byte(" workload-jwt\n"), 0600)).To(Succeed())
			var calls atomic.Int32
			server := tokenServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				want := map[string]string{
					"grant_type":           "urn:ietf:params:oauth:grant-type:token-exchange",
					"requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
					"subject_token":        "workload-jwt", "subject_token_type": options.JWTTokenType,
				}
				if source == "actor" {
					want["subject_token"], want["subject_token_type"] = "target-id", "urn:example:identity"
					want["actor_token"], want["actor_token_type"] = "workload-jwt", "urn:example:assertion"
				}
				for key, value := range want {
					if r.Form.Get(key) != value {
						t.Errorf("%s = %q, want %q", key, r.Form.Get(key), value)
					}
				}
				if r.Method != http.MethodPost || len(r.Form) != len(want) {
					t.Errorf("unexpected exchange request: %s %v", r.Method, r.Form)
				}
				_, _ = fmt.Fprint(w, `{"access_token":"ticket","token_type":"Bearer","expires_in":3600}`)
			})
			config := options.AuthOAuth2TokenExchange{TokenEndpoint: server.URL, SubjectTokenFile: path}
			if source == "literal" {
				config.SubjectTokenFile = ""
				config.SubjectToken, config.SubjectTokenType = "workload-jwt", options.JWTTokenType
			}
			if source == "actor" {
				config.SubjectTokenFile = ""
				config.SubjectToken, config.SubjectTokenType = "target-id", "urn:example:identity"
				config.ActorTokenFile, config.ActorTokenType = path, "urn:example:assertion"
			}
			// Exercise the lazy factory used by both CMS and discovery.
			p := New(&command.BaseOptions{Auth: options.AuthOptions{Type: options.OAuth2TokenExchange, Creds: &config}}, nil, zap.NewNop().Sugar(), nil)
			var wg sync.WaitGroup
			errs := make(chan error, 32)
			for n := 0; n < cap(errs); n++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel, err := p.ContextWithAuth(context.Background())
					if err == nil {
						defer cancel()
						md, _ := metadata.FromOutgoingContext(ctx)
						if tickets := md.Get("x-ydb-auth-ticket"); len(tickets) != 1 || tickets[0] != "Bearer ticket" {
							err = fmt.Errorf("unexpected tickets: %v", tickets)
						}
					}
					errs <- err
				}()
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(calls.Load()).To(Equal(int32(1)))
			g.Expect(os.Remove(path)).To(Succeed())
			for n := 0; n < 32; n++ {
				token, err := p.GetToken(context.Background())
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(token).To(Equal("Bearer ticket"))
			}
			g.Expect(calls.Load()).To(Equal(int32(1)))
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := p.GetToken(ctx)
			g.Expect(err).To(MatchError(context.Canceled))
		})
	}
}

func TestOAuth2Rotation(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%v", symlink), func(t *testing.T) {
			g := NewWithT(t)
			dir := t.TempDir()
			path := filepath.Join(dir, "token")
			first, second := path, filepath.Join(dir, "replacement")
			if symlink {
				first = filepath.Join(dir, "first")
				g.Expect(os.Symlink(first, path)).To(Succeed())
			}
			g.Expect(os.WriteFile(first, []byte("first-jwt"), 0600)).To(Succeed())
			var calls atomic.Int32
			server := tokenServer(t, func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				field := "subject_token"
				if symlink {
					field = "actor_token"
				}
				want, ttl := "first-jwt", 2
				if n > 1 {
					want, ttl = "second-jwt", 3600
				}
				if r.Form.Get(field) != want {
					t.Errorf("exchange %d used %q, want %q", n, r.Form.Get(field), want)
				}
				_, _ = fmt.Fprintf(w, `{"access_token":"ticket-%d","token_type":"Bearer","expires_in":%d}`, n, ttl)
			})
			config := options.AuthOAuth2TokenExchange{TokenEndpoint: server.URL, SubjectTokenFile: path}
			if symlink {
				config.SubjectTokenFile = ""
				config.SubjectToken, config.SubjectTokenType, config.ActorTokenFile = "target-id", "urn:example:identity", path
			}
			p, err := NewOAuth2TokenExchange(config)
			g.Expect(err).NotTo(HaveOccurred())
			token, err := p.GetToken(context.Background())
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(token).To(Equal("Bearer ticket-1"))
			g.Expect(os.WriteFile(second, []byte("second-jwt"), 0600)).To(Succeed())
			if symlink {
				next := filepath.Join(dir, "next-link")
				g.Expect(os.Symlink(second, next)).To(Succeed())
				g.Expect(os.Rename(next, path)).To(Succeed())
			} else {
				g.Expect(os.Rename(second, path)).To(Succeed())
			}
			g.Eventually(func() (string, error) { return p.GetToken(context.Background()) }, 4*time.Second, 20*time.Millisecond).
				Should(Equal("Bearer ticket-2"))
			g.Expect(calls.Load()).To(Equal(int32(2)))
		})
	}
}

func TestOAuth2AcquisitionErrors(t *testing.T) {
	t.Run("exchange honors cancellation", func(t *testing.T) {
		g := NewWithT(t)
		started := make(chan struct{})
		release := make(chan struct{})
		defer close(release)
		server := tokenServer(t, func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
		})
		p, err := NewOAuth2TokenExchange(options.AuthOAuth2TokenExchange{
			TokenEndpoint: server.URL, SubjectToken: "identity", SubjectTokenType: "urn:example:identity",
		})
		g.Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := p.GetToken(ctx)
			done <- err
		}()
		g.Eventually(started, time.Second).Should(BeClosed())
		cancel()
		var acquisitionErr error
		g.Eventually(done, time.Second).Should(Receive(&acquisitionErr))
		g.Expect(errors.Is(acquisitionErr, context.Canceled)).To(BeTrue(), "%v", acquisitionErr)
	})
	for _, body := range []string{
		`{"error":"invalid_grant"}`, `not json`,
		`{"access_token":"ticket","token_type":"Bearer","expires_in":0}`,
		`{"token_type":"Bearer","expires_in":3600}`,
		`{"access_token":"ticket","token_type":"unknown","expires_in":3600}`,
	} {
		t.Run(body, func(t *testing.T) {
			g := NewWithT(t)
			server := tokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
				if body == `{"error":"invalid_grant"}` {
					w.WriteHeader(http.StatusUnauthorized)
				}
				_, _ = fmt.Fprint(w, body)
			})
			p, err := NewOAuth2TokenExchange(options.AuthOAuth2TokenExchange{
				TokenEndpoint: server.URL, SubjectToken: "identity", SubjectTokenType: "urn:example:identity",
			})
			g.Expect(err).NotTo(HaveOccurred())
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			out, authCancel, err := p.ContextWithAuth(ctx)
			g.Expect(err).To(HaveOccurred())
			g.Expect(out).To(BeNil())
			g.Expect(authCancel).To(BeNil())
		})
	}
	t.Run("file errors prevent exchange", func(t *testing.T) {
		g := NewWithT(t)
		var calls atomic.Int32
		server := tokenServer(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
		path := filepath.Join(t.TempDir(), "missing")
		p, err := NewOAuth2TokenExchange(options.AuthOAuth2TokenExchange{TokenEndpoint: server.URL, SubjectTokenFile: path})
		g.Expect(err).NotTo(HaveOccurred())
		_, err = p.GetToken(context.Background())
		g.Expect(err).To(MatchError(ContainSubstring("read OAuth2 token file")))
		g.Expect(os.WriteFile(path, []byte(" \n"), 0600)).To(Succeed())
		_, err = p.GetToken(context.Background())
		g.Expect(err).To(MatchError(ContainSubstring("is empty")))
		g.Expect(calls.Load()).To(BeZero())
	})
	t.Run("failed refresh never returns an expired ticket", func(t *testing.T) {
		g := NewWithT(t)
		server := tokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"access_token":"ticket","token_type":"Bearer","expires_in":1}`)
		})
		path := filepath.Join(t.TempDir(), "token")
		g.Expect(os.WriteFile(path, []byte("jwt"), 0600)).To(Succeed())
		p, err := NewOAuth2TokenExchange(options.AuthOAuth2TokenExchange{TokenEndpoint: server.URL, SubjectTokenFile: path})
		g.Expect(err).NotTo(HaveOccurred())
		_, err = p.GetToken(context.Background())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(os.Remove(path)).To(Succeed())
		token, err := p.GetToken(context.Background())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(token).To(Equal("Bearer ticket"))
		g.Eventually(func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			token, err := p.GetToken(ctx)
			return err != nil && token == ""
		}, 3*time.Second, 20*time.Millisecond).Should(BeTrue())
	})
}
