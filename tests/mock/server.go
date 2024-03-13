package mock

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Auth_V1"
	"github.com/ydb-platform/ydb-go-genproto/Ydb_Cms_V1"
	"github.com/ydb-platform/ydb-go-genproto/Ydb_Discovery_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/Ydb_Maintenance_V1"
	. "github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	TestUser             = "test-user"
	TestPassword         = "test-password"
	TokenForTestUsername = "test-token-here-you-go"
)

type fakeMaintenanceTask struct {
	options           *MaintenanceTaskOptions
	actionGroups      []*ActionGroup
	actionGroupStates []*ActionGroupStates
}

type YdbMock struct {
	Ydb_Maintenance_V1.UnimplementedMaintenanceServiceServer
	Ydb_Cms_V1.UnimplementedCmsServiceServer
	Ydb_Auth_V1.UnimplementedAuthServiceServer
	Ydb_Discovery_V1.UnimplementedDiscoveryServiceServer

	grpcServer *grpc.Server
	caFile     string
	keyFile    string

	// This field contains the list of Nodes that is suitable to return
	// to ListClusterNodes request from rolling restart.
	nodes []*Node
	// This is a layout of the cluster - for example, if the cluster has
	// block-4-2 erasure type and contains 8 storage nodes, then it has
	// only one group: {1, 2, 3, 4, 5, 6, 7, 8}.
	nodeGroups [][]uint32

	// This is just a log of all requests that rolling-restart has sent to
	// the CMS. It is populated during the test and then used only once to
	// check against the expected messages.
	RequestLog []proto.Message

	// All the following fields change during CMS lifetime:

	tasks map[string]*fakeMaintenanceTask
	// These two fields are just 'indexes', they can be calculated from `tasks`
	// but are used for convenience in CMS logic.
	isNodeCurrentlyPermitted map[uint32]bool
	actionToActionUid        map[*Action]*ActionUid
}

func makeSuccessfulOperation() *Ydb_Operations.Operation {
	return &Ydb_Operations.Operation{
		Ready:  true,
		Status: Ydb.StatusIds_SUCCESS,
	}
}

func makeFaultyOperation() *Ydb_Operations.Operation {
	return &Ydb_Operations.Operation{
		Ready:  true,
		Status: Ydb.StatusIds_BAD_REQUEST,
	}
}

func wrapIntoOperation(result proto.Message) *Ydb_Operations.Operation {
	op := makeSuccessfulOperation()
	marshalledResult, _ := anypb.New(result)
	op.Result = marshalledResult
	return op
}

func (s *YdbMock) ListClusterNodes(ctx context.Context, req *ListClusterNodesRequest) (*ListClusterNodesResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	result := &ListClusterNodesResult{
		Nodes: s.nodes,
	}
	return &ListClusterNodesResponse{Operation: wrapIntoOperation(result)}, nil
}

func (s *YdbMock) CreateMaintenanceTask(ctx context.Context, req *CreateMaintenanceTaskRequest) (*MaintenanceTaskResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	taskUid := req.TaskOptions.TaskUid
	s.tasks[taskUid] = &fakeMaintenanceTask{
		options:           req.TaskOptions,
		actionGroups:      req.ActionGroups,
		actionGroupStates: s.makeGroupStatesFor(req.TaskOptions, req.ActionGroups),
	}

	result := &MaintenanceTaskResult{
		TaskUid:           taskUid,
		ActionGroupStates: s.tasks[taskUid].actionGroupStates,
	}
	return &MaintenanceTaskResponse{Operation: wrapIntoOperation(result)}, nil
}

func (s *YdbMock) DropMaintenanceTask(ctx context.Context, req *DropMaintenanceTaskRequest) (*ManageMaintenanceTaskResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	// When we drop a task, some other tasks's actions can be turned into PERFORMED.
	// We don't recalculate all actions states right now, we'll do it on demand, when the
	// client comes for the particular task. For now, we only actualize our helper indexes:
	for _, ag := range s.tasks[req.TaskUid].actionGroups {
		for _, action := range ag.Actions {
			delete(s.actionToActionUid, action)
			actionNodeId := action.GetLockAction().Scope.GetNodeId()
			s.isNodeCurrentlyPermitted[actionNodeId] = false
		}
	}
	delete(s.tasks, req.TaskUid)

	return &ManageMaintenanceTaskResponse{Operation: makeSuccessfulOperation()}, nil
}

func (s *YdbMock) ListMaintenanceTasks(ctx context.Context, req *ListMaintenanceTasksRequest) (*ListMaintenanceTasksResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	taskUids := []string{}
	// Note that we don't calculate anything in this request. ListMaintenanceTasks is very simple -
	// it returns the existing task uids, nothing more. Actual content of each task is not returned.
	for task := range s.tasks {
		taskUids = append(taskUids, task)
	}
	result := &ListMaintenanceTasksResult{
		TasksUids: taskUids,
	}
	return &ListMaintenanceTasksResponse{Operation: wrapIntoOperation(result)}, nil
}

// TODO might need to implement later, but it is not used in rolling restart now, so why bother
// func (s *YdbMock) GetMaintenanceTask(ctx context.Context, req *GetMaintenanceTaskRequest) (*GetMaintenanceTaskResponse, error) {
//
// }

func (s *YdbMock) CompleteAction(ctx context.Context, req *CompleteActionRequest) (*ManageActionResponse, error) {
	s.RequestLog = append(s.RequestLog, req)

	actionStatuses := []*ManageActionResult_Status{}
	for _, completedActionUid := range req.ActionUids {
		s.cleanupActionById(completedActionUid.ActionId)
		actionStatuses = append(actionStatuses, &ManageActionResult_Status{
			ActionUid: completedActionUid,
			Status:    Ydb.StatusIds_SUCCESS,
		})
	}

	result := &ManageActionResult{
		ActionStatuses: actionStatuses,
	}
	return &ManageActionResponse{Operation: wrapIntoOperation(result)}, nil
}

func (s *YdbMock) RefreshMaintenanceTask(ctx context.Context, req *RefreshMaintenanceTaskRequest) (*MaintenanceTaskResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	s.refreshStatesForTask(req.TaskUid)
	result := &MaintenanceTaskResult{
		TaskUid:           req.TaskUid,
		ActionGroupStates: s.tasks[req.TaskUid].actionGroupStates,
		RetryAfter:        timestamppb.New(time.Now().Add(time.Minute * 3)),
	}

	return &MaintenanceTaskResponse{Operation: wrapIntoOperation(result)}, nil
}

func (s *YdbMock) WhoAmI(ctx context.Context, req *Ydb_Discovery.WhoAmIRequest) (*Ydb_Discovery.WhoAmIResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	// TODO check that the grpc authentication header contains the token equal to TokenForTestUsername
	// maybe adding this as the third arg to this function will help: opts ...grpc.CallOption
	result := &Ydb_Discovery.WhoAmIResult{
		User:   TestUser,
		Groups: []string{},
	}
	return &Ydb_Discovery.WhoAmIResponse{Operation: wrapIntoOperation(result)}, nil
}

func (s *YdbMock) Login(ctx context.Context, req *Ydb_Auth.LoginRequest) (*Ydb_Auth.LoginResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	if req.Password == TestPassword && req.User == TestUser {
		result := &Ydb_Auth.LoginResult{
			Token: TokenForTestUsername,
		}
		return &Ydb_Auth.LoginResponse{Operation: wrapIntoOperation(result)}, nil
	}

	return &Ydb_Auth.LoginResponse{Operation: makeFaultyOperation()}, fmt.Errorf("Incorrect credentials")
}

func (s *YdbMock) ListDatabases(ctx context.Context, req *Ydb_Cms.ListDatabasesRequest) (*Ydb_Cms.ListDatabasesResponse, error) {
	s.RequestLog = append(s.RequestLog, req)
	result := &Ydb_Cms.ListDatabasesResult{
		Paths: []string{},
	}

	return &Ydb_Cms.ListDatabasesResponse{Operation: wrapIntoOperation(result)}, nil
}

func NewYdbMockServer() *YdbMock {
	server := &YdbMock{
		tasks:                    make(map[string]*fakeMaintenanceTask),
		actionToActionUid:        make(map[*Action]*ActionUid),
		nodes:                    nil, // filled by initNodes()
		nodeGroups:               nil, // filled by initNodes()
		isNodeCurrentlyPermitted: nil, // filled by initNodes()
	}
	server.initNodes()

	return server
}

func (s *YdbMock) SetupSimpleTLS(caFile, keyFile string) {
	s.caFile = caFile
	s.keyFile = keyFile
}

func (s *YdbMock) StartOn(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if s.caFile != "" && s.keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(s.caFile, s.keyFile)

		if err != nil {
			log.Fatal(err)
		}

		s.grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		s.grpcServer = grpc.NewServer()
	}

	Ydb_Maintenance_V1.RegisterMaintenanceServiceServer(s.grpcServer, s)
	Ydb_Auth_V1.RegisterAuthServiceServer(s.grpcServer, s)
	Ydb_Discovery_V1.RegisterDiscoveryServiceServer(s.grpcServer, s)
	Ydb_Cms_V1.RegisterCmsServiceServer(s.grpcServer, s)

	log.Printf("server listening at %v", lis.Addr())

	go func() {
		if err := s.grpcServer.Serve(lis); err == nil {
			log.Printf("grpc server exited gracefully")
		} else {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
}

func (s *YdbMock) Teardown() {
	s.grpcServer.GracefulStop()
}
