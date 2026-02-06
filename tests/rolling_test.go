package tests

import (
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydbops/tests/mock"
)

var _ = Describe("Test Rolling", func() {
	now := time.Now()
	twoNodesStartedEarlier := now.Add(-10 * time.Minute)
	startedFilterValue := now.Add(-5 * time.Minute)

	BeforeEach(RunBeforeEach)
	AfterEach(RunAfterEach)

	DescribeTable("restart", RunTestCase,
		Entry("restart 2 out of 8 nodes, nodes should be determined by --started filter", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				3: {
					StartTime: twoNodesStartedEarlier,
				},
				7: {
					StartTime: twoNodesStartedEarlier,
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: Command{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"--started", fmt.Sprintf("<%s", startedFilterValue.Format(time.RFC3339)),
						"run",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
					},
					expectedRequests: []proto.Message{
						&Ydb_Auth.LoginRequest{
							User:     mock.TestUser,
							Password: mock.TestPassword,
						},
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Cms.ListDatabasesRequest{},
						&Ydb_Discovery.WhoAmIRequest{
							IncludeGroups: false,
						},
						&Ydb_Maintenance.ListMaintenanceTasksRequest{
							User: &mock.TestUser,
						},
						&Ydb_Maintenance.CreateMaintenanceTaskRequest{
							TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
								TaskUid:          "task-uuid-1",
								Description:      "Rolling restart maintenance task",
								AvailabilityMode: Ydb_Maintenance.AvailabilityMode_AVAILABILITY_MODE_STRONG,
							},
							ActionGroups: mock.MakeActionGroupsFromNodeIds(3, 7),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-uuid-1",
									GroupId:  "group-uuid-1",
									ActionId: "action-uuid-1",
								},
							},
						},
						&Ydb_Maintenance.RefreshMaintenanceTaskRequest{
							TaskUid: "task-uuid-1",
						},
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-uuid-1",
									GroupId:  "group-uuid-2",
									ActionId: "action-uuid-2",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("happy path: restart 3 out of 8 nodes, strong mode, no failures", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--hosts=1,2,3",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
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
				},
			},
		},
		),
		Entry("--availability-mode weak should take up to 2 nodes from 8-node storage group", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "weak",
						"--hosts=1,2",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
								AvailabilityMode: Ydb_Maintenance.AvailabilityMode_AVAILABILITY_MODE_WEAK,
							},
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1, 2),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("happy path: restart 3 out of 3 nodes, no --hosts", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
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
				},
			},
		},
		),
		Entry("filter nodes by --version flag, >MAJOR.MINOR.PATCH", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {
					Version: "ydb-stable-24-1-0",
				},
				2: {
					Version: "ydb-stable-24-2-0",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--version", ">24.1.0",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(2),
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
					},
				},
			},
		},
		),
		Entry("filter nodes by --version flag, ==full_version_string_1234", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {
					Version: "full_version_string_1234",
				},
				2: {
					Version: "full_version_string_1235",
				},
				3: {
					Version: "full_version_string_1236",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--version", "==full_version_string_1234",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1),
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
					},
				},
			},
		},
		),
		Entry("filter nodes by --version flag, !=full_version_string_1234", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {
					Version: "full_version_string_1234",
				},
				2: {
					Version: "full_version_string_1235",
				},
				3: {
					Version: "full_version_string_1236",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--version", "!=full_version_string_1234",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(2, 3),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("happy path, restart dynnodes by tenant name", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
				{9, 10, 11},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				9: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				10: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
				11: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"--tenant",
						"--tenant-list=fakeTenant2",
						"run",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(10, 11),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("will restart tenants in parallel", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
				{9, 10},
				{11, 12},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				9: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				10: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				11: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
				12: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--tenant",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(9, 10, 11, 12),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-3",
									ActionId: "action-UUID-3",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-4",
									ActionId: "action-UUID-4",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("restart two tenants concurrently with --tenants-inflight=2", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
				{9, 10},
				{11, 12},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				9: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				10: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				11: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
				12: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"--tenants-inflight", "2",
						"--ordering-key", "tenant",
						"run",
						"--tenant",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(9, 10, 11, 12),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-3",
									ActionId: "action-UUID-3",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-4",
									ActionId: "action-UUID-4",
								},
							},
						},
					},
					expectedOutputRegexps: []string{
						"2 ActionGroupStates moved to PERFORMED",
						"calculated batches",
						"dispatching batch",
						"Restart completed successfully",
					},
				},
			},
		},
		),
		Entry("restart multiple tenants and multiple nodes per tenant, both nodes and tenants inflight = 2", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
				{9, 10},
				{11, 12},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				9: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				10: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
				11: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
				12: {
					IsDynnode:  true,
					TenantName: "fakeTenant2",
				},
			},
			additionalTestBehaviour: &mock.AdditionalTestBehaviour{
				MaxDynnodesPerformedPerTenant: 4,
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"--nodes-inflight", "2",
						"--tenants-inflight", "2",
						"--ordering-key", "tenant",
						"run",
						"--tenant",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIdsWithInflight(2, 9, 10, 11, 12),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-1",
									ActionId: "action-UUID-1",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-3",
									ActionId: "action-UUID-3",
								},
								{
									TaskUid:  "task-UUID-1",
									GroupId:  "group-UUID-4",
									ActionId: "action-UUID-4",
								},
							},
						},
					},
					expectedOutputRegexps: []string{
						"4 ActionGroupStates moved to PERFORMED",
						"calculated batches",
						"dispatching batch",
						"Restart completed successfully",
					},
				},
			},
		},
		),
		Entry("do not restart nodes that are down", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {
					State: Ydb_Maintenance.ItemState_ITEM_STATE_DOWN,
				},
				2: {
					State: Ydb_Maintenance.ItemState_ITEM_STATE_DOWN,
				},
				3: {
					State: Ydb_Maintenance.ItemState_ITEM_STATE_MAINTENANCE,
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--storage",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
					},
				},
			},
		},
		),
		Entry("does not restart storage and tenants in parallel (if filters are unspecified)", TestCase{
			nodeConfiguration: [][]uint32{
				{1},
				{2},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				2: {
					IsDynnode:  true,
					TenantName: "fakeTenant1",
				},
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
						&Ydb_Cms.ListDatabasesRequest{},
						&Ydb_Discovery.WhoAmIRequest{},
						&Ydb_Maintenance.ListMaintenanceTasksRequest{
							User: &mock.TestUser,
						},
						&Ydb_Maintenance.CreateMaintenanceTaskRequest{
							TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
								TaskUid:          "task-UUID-2",
								Description:      "Rolling restart maintenance task",
								AvailabilityMode: Ydb_Maintenance.AvailabilityMode_AVAILABILITY_MODE_STRONG,
							},
							ActionGroups: mock.MakeActionGroupsFromNodeIds(2),
						},
						&Ydb_Maintenance.CompleteActionRequest{
							ActionUids: []*Ydb_Maintenance.ActionUid{
								{
									TaskUid:  "task-UUID-2",
									GroupId:  "group-UUID-2",
									ActionId: "action-UUID-2",
								},
							},
						},
					},
				},
			},
		},
		),
		Entry("Disallow more than 2 major-minor combinations in the cluster", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
				{4},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {
					IsDynnode: false,
					Version:   "23.4.1",
				},
				2: {
					IsDynnode: false,
					Version:   "23.4.1",
				},
				3: {
					IsDynnode: false,
					Version:   "23.4.1",
				},
				4: {
					IsDynnode:  true,
					TenantName: "fakeTenant",
					Version:    "23.3.1",
				},
			},
			additionalTestBehaviour: &mock.AdditionalTestBehaviour{
				RestartNodesOnNewVersion: "24.1.1",
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--hosts", "1,2",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1, 2),
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
						&Ydb_Maintenance.ListClusterNodesRequest{},
					},
					expectedOutputRegexps: []string{
						".*Triggered this check: 24 major is incompatible with 23-3.*",
					},
				},
			},
		},
		),
		Entry("Interrupt rolling restart with SIGTERM", TestCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{
				1: {IsDynnode: false},
				2: {IsDynnode: false},
				3: {IsDynnode: false},
			},
			additionalTestBehaviour: &mock.AdditionalTestBehaviour{
				SignalDelayMs: 500, // this is carefully crafted to fire just when `ydbops` waits for second node
			},
			steps: []StepData{
				{
					ydbopsInvocation: []string{
						"--endpoint", "grpcs://localhost:2135",
						"--verbose",
						"--availability-mode", "strong",
						"--user", mock.TestUser,
						"--cms-query-interval", "1",
						"run",
						"--hosts", "1,2",
						"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
						"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
						"--cleanup-on-exit",
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
							ActionGroups: mock.MakeActionGroupsFromNodeIds(1, 2),
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
						&Ydb_Maintenance.ListMaintenanceTasksRequest{
							User: &mock.TestUser,
						},
						&Ydb_Maintenance.GetMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
						&Ydb_Maintenance.DropMaintenanceTaskRequest{
							TaskUid: "task-UUID-1",
						},
					},
					expectedOutputRegexps: []string{},
				},
			},
		},
		),
	)
})
