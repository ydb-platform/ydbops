package connectionsfactory

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/pkg/command"
)

const (
	BufferSize = 32 << 20

	// This gets added on top of OperationTimeout, so grpc call can terminate
	// if any transport layer errors occur. Without this timeout, we wait forever
	// inside retry loop, not completing a single request
	DefaultTransportTimeout = 10 * time.Second
)

type Factory interface {
	Create() (*grpc.ClientConn, error)
	OperationParams() *Ydb_Operations.OperationParams
	OperationTimeout() time.Duration
	CallTimeout() time.Duration
}

func New(
	options *command.BaseOptions,
	transportTimeout time.Duration,
) Factory {
	return &connectionsFactory{
		options:          options,
		transportTimeout: transportTimeout,
	}
}

type connectionsFactory struct {
	options          *command.BaseOptions
	transportTimeout time.Duration
}

func (f *connectionsFactory) OperationParams() *Ydb_Operations.OperationParams {
	return &Ydb_Operations.OperationParams{
		OperationMode:    Ydb_Operations.OperationParams_SYNC,
		OperationTimeout: durationpb.New(f.OperationTimeout()),
		CancelAfter:      durationpb.New(f.OperationTimeout()),
	}
}

func (f *connectionsFactory) OperationTimeout() time.Duration {
	return time.Duration(f.options.GRPC.TimeoutSeconds) * time.Second
}

func (f *connectionsFactory) CallTimeout() time.Duration {
	return f.OperationTimeout() + f.transportTimeout
}

func (f *connectionsFactory) Create() (*grpc.ClientConn, error) {
	cr, err := f.makeCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	if f.options.GRPC.Endpoint == "" {
		return nil, fmt.Errorf("specify a grpc endpoint with --endpoint")
	}

	return grpc.Dial(f.endpoint(),
		grpc.WithTransportCredentials(cr),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(BufferSize),
			grpc.MaxCallRecvMsgSize(BufferSize)))
}

func (f *connectionsFactory) endpoint() string {
	// TODO decide if we want to support multiple endpoints or just one
	// Endpoint in rootOpts will turn from string -> []string in this case
	//
	// for balancers, it does not really matter, one endpoint is enough.
	// but if you specify node endpoint directly, if this particular node
	// is dead, things get inconvenient.
	return fmt.Sprintf("%s:%d", f.options.GRPC.Endpoint, f.options.GRPC.GRPCPort)
}

func (f *connectionsFactory) makeCredentials() (credentials.TransportCredentials, error) {
	if !f.options.GRPC.GRPCSecure {
		return insecure.NewCredentials(), nil
	}

	systemPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get the system cert pool: %w", err)
	}

	if f.options.GRPC.CaFile != "" {
		b, err := os.ReadFile(f.options.GRPC.CaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read the ca file: %w", err)
		}
		if !systemPool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("credentials: failed to append certificates")
		}
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    systemPool,
	}

	if f.options.GRPC.GRPCSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return credentials.NewTLS(tlsConfig), nil
}
