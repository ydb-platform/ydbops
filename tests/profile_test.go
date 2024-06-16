package tests

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydbops/tests/mock"
)

var _ = Describe("Test Profile", func() {
	BeforeEach(RunBeforeEach)
	AfterEach(RunAfterEach)

	DescribeTable("profile", RunTestCase,
		Entry("some basic options, no --profile option, active_profile in config", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			steps: []StepData{
				{
					ydbopsInvocation: Command{
						"--config-file",
						filepath.Join(".", "test-data", "config_with_active_profile.yaml"),
						"--availability-mode", "strong",
						"--cms-query-interval", "1",
						"run",
						"--hosts=1,2,3",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
					},
					expectedRequests: []proto.Message{
						&Ydb_Auth.LoginRequest{
							User:     mock.TestUser,
							Password: mock.TestPassword,
						},
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Cms.ListDatabasesRequest{},
						&Ydb_Discovery.WhoAmIRequest{},
						&Ydb_Maintenance.ListMaintenanceTasksRequest{
							User: &mock.TestUser,
						},
						&Ydb_Maintenance.CreateMaintenanceTaskRequest{
							TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
								TaskUid:          "task-UUID-1",
								Description:      "Rolling restart maintenance task",
								AvailabilityMode: Ydb_Maintenance.AvailabilityMode_AVAILABILITY_MODE_STRONG,
							},
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1, 2, 3),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
							},
						},
						&Ydb_Maintenance.RefreshMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
						&Ydb_Maintenance.RefreshMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-3",
									ActionId: "action-UUID-3",
								},
							},
						},
					},
					expectedOutputRegexps: []string{},
				},
			},
		},
		),
		Entry("some basic options, --profile option specified, no active_profile in config", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			steps: []StepData{
				{
					ydbopsInvocation: Command{
						"--config-file",
						filepath.Join(".", "test-data", "config_without_active_profile.yaml"),
						"--profile",
						"my-profile",
						"--availability-mode", "strong",
						"--cms-query-interval", "1",
						"run",
						"--hosts=1,2,3",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
					},
					expectedRequests: []proto.Message{
						&Ydb_Auth.LoginRequest{
							User:     mock.TestUser,
							Password: mock.TestPassword,
						},
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Cms.ListDatabasesRequest{},
						&Ydb_Discovery.WhoAmIRequest{},
						&Ydb_Maintenance.ListMaintenanceTasksRequest{
							User: &mock.TestUser,
						},
						&Ydb_Maintenance.CreateMaintenanceTaskRequest{
							TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
								TaskUid:          "task-UUID-1",
								Description:      "Rolling restart maintenance task",
								AvailabilityMode: Ydb_Maintenance.AvailabilityMode_AVAILABILITY_MODE_STRONG,
							},
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1, 2, 3),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
							},
						},
						&Ydb_Maintenance.RefreshMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
						&Ydb_Maintenance.RefreshMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-3",
									ActionId: "action-UUID-3",
								},
							},
						},
					},
					expectedOutputRegexps: []string{},
				},
			},
		},
		),
	)
})
