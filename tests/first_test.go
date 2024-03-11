package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Auth"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	blackmagic "github.com/ydb-platform/ydbops/tests/black-magic"
	"github.com/ydb-platform/ydbops/tests/mock"
	"google.golang.org/protobuf/proto"
)

func setupYdbMock(port int) *mock.YdbMock {
	ydbMock := mock.NewYdbMockServer()
	ydbMock.StartOn(port)
	return ydbMock
}

func prepareEnvVariables() map[string]string {
	previous := make(map[string]string)

	newValue := mock.TestPassword
	os.Setenv("YDB_PASSWORD", newValue)
	previous["YDB_PASSWORD"] = os.Getenv("YDB_PASSWORD")

	return previous
}

func revertEnvVariables(previous map[string]string) {
	for k, v := range previous {
		os.Setenv(k, v)
	}
}

var _ = Describe("Test Rolling", func() {
	It("my first rolling emulation", func() {
		port := 2135
		cms := setupYdbMock(port)
		defer cms.Teardown()

		previousEnvVars := prepareEnvVariables()
		defer revertEnvVariables(previousEnvVars)

		cmd := exec.Command(filepath.Join("..", "ydbops"),
			"--endpoint", "grpc://localhost:2135",
			"--verbose",
			"restart",
			"--availability-mode", "strong",
			"--hosts=1,2,3",
			"--user", mock.TestUser,
			"run",
			"--payload", filepath.Join(".", "mock", "noop-payload.sh"),
		)

		_, err := cmd.CombinedOutput()
		// fmt.Println(string(output))

		Expect(err).To(BeNil())

		if err != nil {
			fmt.Println("Error getting combined output:", err)
			return
		}

		expectedRequests := []proto.Message{
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
				ActionGroups: []*Ydb_Maintenance.ActionGroup{
					{}, {}, {},
				},
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
		}

		actualRequests := cms.RequestLog

		// for _, req := range actualRequests {
		// 	fmt.Printf("\n%+v : %+v\n", reflect.TypeOf(req), req)
		// }

		Expect(len(expectedRequests)).To(Equal(len(actualRequests)))

		values := make(map[string]string)
		for i, expected := range expectedRequests {
			actual := actualRequests[i]
			blackmagic.DeepEqualOnPresentFields(expected, actual, values)
		}
	})
})
