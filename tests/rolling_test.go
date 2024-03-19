package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/ydb-platform/ydbops/pkg/options"
	blackmagic "github.com/ydb-platform/ydbops/tests/black-magic"
	"github.com/ydb-platform/ydbops/tests/mock"
)

func prepareEnvVariables() map[string]string {
	previous := make(map[string]string)

	newValue := mock.TestPassword
	os.Setenv(options.DefaultStaticPasswordEnvVar, newValue)
	previous[options.DefaultStaticPasswordEnvVar] = os.Getenv(options.DefaultStaticPasswordEnvVar)

	return previous
}

func revertEnvVariables(previous map[string]string) {
	for k, v := range previous {
		os.Setenv(k, v)
	}
}

var _ = Describe("Test Rolling", func() {
	var ydb *mock.YdbMock
	var previousEnvVars map[string]string

	now := time.Now()
	twoNodesStartedEarlier := now.Add(-10 * time.Minute)
	startedFilterValue := now.Add(-5 * time.Minute)

	type testCase struct {
		nodeConfiguration [][]uint32
		nodeInfoMap       map[uint32]mock.TestNodeInfo
		expectedRequests  []proto.Message
		ydbopsInvocation  []string
	}

	BeforeEach(func() {
		port := 2135
		ydb = mock.NewYdbMockServer()
		ydb.SetupSimpleTLS(
			filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
			filepath.Join(".", "test-data", "ssl-data", "ca_unencrypted.key"),
		)
		ydb.StartOn(port)

		previousEnvVars = prepareEnvVariables()
	})

	AfterEach(func() {
		ydb.Teardown()
		revertEnvVariables(previousEnvVars)
	})

	DescribeTable("restart", func(tc testCase) {
		ydb.SetNodeConfiguration(tc.nodeConfiguration, tc.nodeInfoMap)

		cmd := exec.Command(filepath.Join("..", "ydbops"), tc.ydbopsInvocation...)

		_, err := cmd.CombinedOutput()
		// output, err := cmd.CombinedOutput()
		// fmt.Println(string(output))
		Expect(err).To(BeNil())

		if err != nil {
			fmt.Println("Error getting combined output:", err)
			return
		}

		actualRequests := ydb.RequestLog

		// for _, req := range actualRequests {
		// 	fmt.Printf("\n%+v : %+v\n", reflect.TypeOf(req), req)
		// }

		for _, actualReq := range actualRequests {
			field := reflect.ValueOf(actualReq).Elem().FieldByName("OperationParams")
			if field.IsValid() {
				field.Set(reflect.Zero(field.Type()))
			}
		}

		defer func() {
			if r := recover(); r != nil {
				if strings.Contains(fmt.Sprintf("%v", r), "non-deterministic or non-symmetric function detected") {
					Fail(`UuidComparer failed, see logs for more info.`)
				} else {
					panic(r)
				}
			}
		}()

		expectedPlaceholders := make(map[string]int)
		actualPlaceholders := make(map[string]int)

		for i, expected := range tc.expectedRequests {
			actual := actualRequests[i]
			Expect(cmp.Diff(expected, actual,
				protocmp.Transform(),
				blackmagic.ActionGroupSorter(),
				blackmagic.UuidComparer(expectedPlaceholders, actualPlaceholders),
			)).To(BeEmpty())
		}

		Expect(len(tc.expectedRequests)).To(Equal(len(actualRequests)))
	},
		Entry("restart 2 out of 8 nodes, nodes should be determined by --started filter", testCase{
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
			ydbopsInvocation: []string{
				"--endpoint", "grpcs://localhost:2135",
				"--verbose",
				"restart",
				"--availability-mode", "strong",
				"--user", mock.TestUser,
				"--cms-query-interval", "1",
				"--started", fmt.Sprintf("<%s", startedFilterValue.Format(time.RFC3339)),
				"run",
				"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
				"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
			},
			expectedRequests: []proto.Message{
				&Ydb_Auth.LoginRequest{
					User:     mock.TestUser,
					Password: mock.TestPassword,
				},
				&Ydb_Cms.ListDatabasesRequest{},
				&Ydb_Maintenance.ListClusterNodesRequest{},
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
					ActionGroups: mock.MakeActionGroups(3, 7),
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
		),
		Entry("happy path: restart 3 out of 8 nodes, strong mode, no failures", testCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3, 4, 5, 6, 7, 8},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			ydbopsInvocation: []string{
				"--endpoint", "grpcs://localhost:2135",
				"--verbose",
				"restart",
				"--availability-mode", "strong",
				"--hosts=1,2,3",
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
				&Ydb_Cms.ListDatabasesRequest{},
				&Ydb_Maintenance.ListClusterNodesRequest{},
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
					ActionGroups: mock.MakeActionGroups(1, 2, 3),
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
		},
		),
		Entry("happy path: restart 3 out of 3 nodes, no --hosts", testCase{
			nodeConfiguration: [][]uint32{
				{1, 2, 3},
			},
			nodeInfoMap: map[uint32]mock.TestNodeInfo{},
			ydbopsInvocation: []string{
				"--endpoint", "grpcs://localhost:2135",
				"--verbose",
				"restart",
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
				&Ydb_Cms.ListDatabasesRequest{},
				&Ydb_Maintenance.ListClusterNodesRequest{},
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
					ActionGroups: mock.MakeActionGroups(1, 2, 3),
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
		},
		),
		Entry("filter nodes by --version flag", testCase{
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
				3: {
					Version: "ydb-stable-24-3-0",
				},
			},
			ydbopsInvocation: []string{
				"--endpoint", "grpcs://localhost:2135",
				"--verbose",
				"restart",
				"--availability-mode", "strong",
				"--user", mock.TestUser,
				"--cms-query-interval", "1",
				"run",
				"--version", ">24.1.0",
				"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
				"--ca-file", filepath.Join(".", "test-data", "ssl-data", "ca.crt"),
			},
			expectedRequests: []proto.Message{
				&Ydb_Auth.LoginRequest{
					User:     mock.TestUser,
					Password: mock.TestPassword,
				},
				&Ydb_Cms.ListDatabasesRequest{},
				&Ydb_Maintenance.ListClusterNodesRequest{},
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
					ActionGroups: mock.MakeActionGroups(2, 3),
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
			},
		},
		),
	)
})
