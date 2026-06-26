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

type Config struct {
	Endpoint         string
	GRPCPort         int
	GRPCSecure       bool
	GRPCSkipVerify   bool
	CaFile           string
	OperationTimeout time.Duration
	TransportTimeout time.Duration
}

func NewFromConfig(config Config) Factory {
	return NewFromDelayedConfig(func() Config {
		return config
	})
}

func NewFromDelayedConfig(configProvider func() Config) Factory {
	return &connectionsFactory{configProvider: configProvider}
}

type connectionsFactory struct {
	configProvider func() Config
}

func (f *connectionsFactory) OperationParams() *Ydb_Operations.OperationParams {
	return &Ydb_Operations.OperationParams{
		OperationMode:    Ydb_Operations.OperationParams_SYNC,
		OperationTimeout: durationpb.New(f.OperationTimeout()),
		CancelAfter:      durationpb.New(f.OperationTimeout()),
	}
}

func (f *connectionsFactory) OperationTimeout() time.Duration {
	return f.configProvider().OperationTimeout
}

func (f *connectionsFactory) CallTimeout() time.Duration {
	config := f.configProvider()
	return config.OperationTimeout + config.TransportTimeout
}

func (f *connectionsFactory) Create() (*grpc.ClientConn, error) {
	config := f.configProvider()

	cr, err := makeCredentials(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("specify a grpc endpoint with --endpoint")
	}

	return grpc.Dial(endpoint(config),
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

func endpoint(config Config) string {
	// TODO decide if we want to support multiple endpoints or just one
	// Endpoint in rootOpts will turn from string -> []string in this case
	//
	// for balancers, it does not really matter, one endpoint is enough.
	// but if you specify node endpoint directly, if this particular node
	// is dead, things get inconvenient.
	return fmt.Sprintf("%s:%d", config.Endpoint, config.GRPCPort)
}

func makeCredentials(config Config) (credentials.TransportCredentials, error) {
	if !config.GRPCSecure {
		return insecure.NewCredentials(), nil
	}

	systemPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get the system cert pool: %w", err)
	}

	if config.CaFile != "" {
		b, err := os.ReadFile(config.CaFile)
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

	if config.GRPCSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return credentials.NewTLS(tlsConfig), nil
}
