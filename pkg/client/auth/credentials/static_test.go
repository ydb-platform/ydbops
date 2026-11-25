package credentials

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type loginFunc func(context.Context, string, string) (string, error)

func (f loginFunc) Auth(ctx context.Context, user, password string) (string, error) {
	return f(ctx, user, password)
}

func TestStaticRetriesFailedLogin(t *testing.T) {
	for _, failure := range []error{context.Canceled, context.DeadlineExceeded, errors.New("temporary login failure")} {
		t.Run(failure.Error(), func(t *testing.T) {
			p := NewStatic("user", "password", nil, nil).(*staticCredentialsProvider)
			var attempts atomic.Int32
			p.authClient = loginFunc(func(context.Context, string, string) (string, error) {
				if attempts.Add(1) == 1 {
					return "", failure
				}
				return "ticket", nil
			})
			if _, err := p.GetToken(context.Background()); !errors.Is(err, failure) {
				t.Fatalf("first login error = %v, want %v", err, failure)
			}

			var wg sync.WaitGroup
			for i := 0; i < 16; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					token, err := p.GetToken(context.Background())
					if err != nil || token != "ticket" {
						t.Errorf("token = %q, error = %v", token, err)
					}
				}()
			}
			wg.Wait()
			if got := attempts.Load(); got != 2 {
				t.Fatalf("login attempts = %d, want one failure and one success", got)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := p.GetToken(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled caller error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestStaticWaitingCallerCancellation(t *testing.T) {
	p := NewStatic("user", "password", nil, nil).(*staticCredentialsProvider)
	started := make(chan struct{})
	var attempts atomic.Int32
	p.authClient = loginFunc(func(ctx context.Context, _, _ string) (string, error) {
		if attempts.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			return "", ctx.Err()
		}
		return "ticket", nil
	})
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	firstDone := make(chan error, 1)
	go func() {
		_, err := p.GetToken(firstCtx)
		firstDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first login never started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := p.GetToken(ctx)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiting caller error = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waiting caller ignored its deadline")
	}

	cancelFirst()
	select {
	case err := <-firstDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("first caller error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first login ignored cancellation")
	}
	if token, err := p.GetToken(context.Background()); err != nil || token != "ticket" {
		t.Fatalf("login after cancellation: token = %q, error = %v", token, err)
	}
}
