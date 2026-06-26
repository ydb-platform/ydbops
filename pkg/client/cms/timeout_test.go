package cms_test

import (
	"context"
	"net"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ydb-platform/ydb-go-genproto/draft/Ydb_Maintenance_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	authprovider "github.com/ydb-platform/ydbops/pkg/client/auth/provider"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
)

type hangingMaintenanceServer struct {
	Ydb_Maintenance_V1.UnimplementedMaintenanceServiceServer
}

func (*hangingMaintenanceServer) CreateMaintenanceTask(
	ctx context.Context,
	_ *Ydb_Maintenance.CreateMaintenanceTaskRequest,
) (*Ydb_Maintenance.MaintenanceTaskResponse, error) {
	// We avoid replying until timeout runs out. From the client perspective,
	// this is equivalent to stuck TCP connection, so we can test L3 level timeout
	// this way.
	<-ctx.Done()
	return nil, ctx.Err()
}

var _ = Describe("CMS client deadline", func() {
	// this test could have been an e2e test as most our tests, but it relies on overriding
	// timeout values for quicker test (still cant use fake clock though), so we need to call
	// functions directly -> integration test instead
	It("aborts a stuck CreateMaintenanceTask at the per-call deadline", func() {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		srv := grpc.NewServer()
		Ydb_Maintenance_V1.RegisterMaintenanceServiceServer(srv, &hangingMaintenanceServer{})
		go func() { _ = srv.Serve(lis) }()
		DeferCleanup(srv.Stop)

		host, portStr, err := net.SplitHostPort(lis.Addr().String())
		Expect(err).NotTo(HaveOccurred())
		port, err := strconv.Atoi(portStr)
		Expect(err).NotTo(HaveOccurred())

		factory := connectionsfactory.NewFromConfig(connectionsfactory.Config{
			Endpoint:         host,
			GRPCPort:         port,
			GRPCSecure:       false,
			OperationTimeout: 0, // this is a YDB OperationTimeout, does not make sense for this test as our mock does not use it
			TransportTimeout: 200 * time.Millisecond,
		})
		creds := authprovider.NewProviderFromToken("fake")
		client := cms.NewCMSClient(factory, zap.NewNop().Sugar(), creds)
		DeferCleanup(client.Close)

		// Setting a global timeout for this test:
		// we have request + 200ms timeout + one second (default) backoff retry + another 200ms timeout
		// this fits in 1.5 seconds nicely
		bounded, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		client.SetContext(bounded)

		// to check if the call is stuck, we need to run the actual test in a
		// separate goroutine and check on deadline expiration
		type callResult struct {
			err     error
			elapsed time.Duration
		}
		done := make(chan callResult, 1)
		start := time.Now()
		go func() {
			_, err := client.CreateMaintenanceTask(cms.MaintenanceTaskParams{
				TaskUID:   "deadline-test",
				ScopeType: cms.NodeScope,
				Nodes:     []*Ydb_Maintenance.Node{{NodeId: 1}},
			})
			done <- callResult{err: err, elapsed: time.Since(start)}
		}()

		select {
		case r := <-done:
			Expect(r.err).To(HaveOccurred())
			GinkgoT().Logf("CreateMaintenanceTask returned %v after %v", r.err, r.elapsed)
		case <-time.After(2 * time.Second):
			Fail("CreateMaintenanceTask expected to return within 1.5 seconds but still didn't return after 2 seconds")
		}
	})
})
