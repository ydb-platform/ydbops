package cms

import (
	"context"
	"fmt"
	"strings"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Cms_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/Ydb_Maintenance_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Issue"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/ydb-platform/ydb-ops/internal/util"
)

type CMSClient struct {
	logger *zap.SugaredLogger
	f      *Factory
}

type operationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}

func NewCMSClient(logger *zap.SugaredLogger, f *Factory) *CMSClient {
	return &CMSClient{
		logger: logger,
		f:      f,
	}
}

func (c *CMSClient) Tenants() ([]string, error) {
	result := Ydb_Cms.ListDatabasesResult{}
	_, err := c.ExecuteCMSMethod(&result, func(ctx context.Context, cl Ydb_Cms_V1.CmsServiceClient) (operationResponse, error) {
		c.logger.Debug("Invoke ListDatabases method")
		return cl.ListDatabases(ctx, &Ydb_Cms.ListDatabasesRequest{OperationParams: c.f.OperationParams()})
	})
	if err != nil {
		return nil, err
	}

	s := util.SortBy(result.Paths,
		func(l string, r string) bool {
			return l < r
		},
	)
	return s, nil
}

func (c *CMSClient) Nodes() ([]*Ydb_Maintenance.Node, error) {
	result := Ydb_Maintenance.ListClusterNodesResult{}
	_, err := c.ExecuteMaintenanceMethod(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke ListClusterNodes method")
			return cl.ListClusterNodes(ctx, &Ydb_Maintenance.ListClusterNodesRequest{OperationParams: c.f.OperationParams()})
		},
	)
	if err != nil {
		return nil, err
	}

	nodes := util.SortBy(result.Nodes,
		func(l *Ydb_Maintenance.Node, r *Ydb_Maintenance.Node) bool {
			return l.NodeId < r.NodeId
		},
	)

	return nodes, nil
}

func (c *CMSClient) MaintenanceTasks() ([]string, error) {
	result := Ydb_Maintenance.ListMaintenanceTasksResult{}
	_, err := c.ExecuteMaintenanceMethod(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke ListMaintenanceTasks method")
			return cl.ListMaintenanceTasks(ctx,
				&Ydb_Maintenance.ListMaintenanceTasksRequest{
					OperationParams: c.f.OperationParams(),
					User:            util.Pointer(c.f.User()),
				},
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return result.TasksUids, nil
}

func (c *CMSClient) GetMaintenanceTask(taskId string) (MaintenanceTask, error) {
	result := Ydb_Maintenance.GetMaintenanceTaskResult{}
	_, err := c.ExecuteMaintenanceMethod(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke GetMaintenanceTask method")
			return cl.GetMaintenanceTask(ctx, &Ydb_Maintenance.GetMaintenanceTaskRequest{
				OperationParams: c.f.OperationParams(),
				TaskUid:         taskId,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return &maintenanceTaskResult{
		TaskUid:           taskId,
		ActionGroupStates: result.ActionGroupStates,
	}, nil
}

func (c *CMSClient) CreateMaintenanceTask(params MaintenanceTaskParams) (MaintenanceTask, error) {
	request := &Ydb_Maintenance.CreateMaintenanceTaskRequest{
		OperationParams: c.f.OperationParams(),
		TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
			TaskUid:          params.TaskUid,
			AvailabilityMode: params.AvailAbilityMode,
			Description:      "Rolling restart maintenance task",
		},
		ActionGroups: make([]*Ydb_Maintenance.ActionGroup, 0, len(params.Nodes)),
	}

	for _, node := range params.Nodes {
		request.ActionGroups = append(request.ActionGroups,
			&Ydb_Maintenance.ActionGroup{
				Actions: []*Ydb_Maintenance.Action{
					{
						Action: &Ydb_Maintenance.Action_LockAction{
							LockAction: &Ydb_Maintenance.LockAction{
								Scope: &Ydb_Maintenance.ActionScope{
									Scope: &Ydb_Maintenance.ActionScope_NodeId{
										NodeId: node.NodeId,
									},
								},
								Duration: params.Duration,
							},
						},
					},
				},
			},
		)
	}

	result := &Ydb_Maintenance.MaintenanceTaskResult{}
	_, err := c.ExecuteMaintenanceMethod(result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke CreateMaintenanceTask method")
			return cl.CreateMaintenanceTask(ctx, request)
		},
	)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *CMSClient) RefreshMaintenanceTask(taskId string) (MaintenanceTask, error) {
	result := Ydb_Maintenance.MaintenanceTaskResult{}
	_, err := c.ExecuteMaintenanceMethod(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke RefreshMaintenanceTask method")
			return cl.RefreshMaintenanceTask(ctx, &Ydb_Maintenance.RefreshMaintenanceTaskRequest{
				OperationParams: c.f.OperationParams(),
				TaskUid:         taskId,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *CMSClient) DropMaintenanceTask(taskId string) (string, error) {
	op, err := c.ExecuteMaintenanceMethod(nil,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke DropMaintenanceTask method")
			return cl.DropMaintenanceTask(ctx, &Ydb_Maintenance.DropMaintenanceTaskRequest{
				OperationParams: c.f.OperationParams(),
				TaskUid:         taskId,
			})
		},
	)
	if err != nil {
		return "", err
	}

	return op.Status.String(), nil
}

func (c *CMSClient) CompleteAction(actionIds []*Ydb_Maintenance.ActionUid) (*Ydb_Maintenance.ManageActionResult, error) {
	result := Ydb_Maintenance.ManageActionResult{}
	op, err := c.ExecuteMaintenanceMethod(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error) {
			c.logger.Debug("Invoke CompleteAction method")
			return cl.CompleteAction(ctx, &Ydb_Maintenance.CompleteActionRequest{
				OperationParams: c.f.OperationParams(),
				ActionUids:      actionIds,
			})
		},
	)
	_ = op
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *CMSClient) ExecuteMaintenanceMethod(
	out proto.Message,
	method func(context.Context, Ydb_Maintenance_V1.MaintenanceServiceClient) (operationResponse, error),
) (*Ydb_Operations.Operation, error) {
	// todo:
	// 	 1, error handling ??
	//   2. retries ??

	cc, err := c.f.Connection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := c.f.Context()
	defer cancel()

	cl := Ydb_Maintenance_V1.NewMaintenanceServiceClient(cc)
	r, err := method(ctx, cl)
	if err != nil {
		c.logger.Debugf("Invocation error: %+v", err)
		return nil, err
	}
	op := r.GetOperation()
	c.logOperation(op)

	if out == nil {
		return op, nil
	}

	if err := op.Result.UnmarshalTo(out); err != nil {
		return op, err
	}

	if op.Status != Ydb.StatusIds_SUCCESS {
		return op, fmt.Errorf("unsuccessful status code: %s", op.Status)
	}

	return op, nil
}

func (c *CMSClient) ExecuteCMSMethod(
	out proto.Message,
	method func(context.Context, Ydb_Cms_V1.CmsServiceClient) (operationResponse, error),
) (*Ydb_Operations.Operation, error) {
	cc, err := c.f.Connection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := c.f.Context()
	defer cancel()

	cl := Ydb_Cms_V1.NewCmsServiceClient(cc)
	r, err := method(ctx, cl)
	if err != nil {
		c.logger.Debugf("Invocation error: %+v", err)
		return nil, err
	}
	op := r.GetOperation()
	c.logOperation(op)

	if out == nil {
		return op, nil
	}

	if err := op.Result.UnmarshalTo(out); err != nil {
		return op, err
	}

	if op.Status != Ydb.StatusIds_SUCCESS {
		return op, fmt.Errorf("unsuccessful status code: %s", op.Status)
	}

	return op, nil
}

func (c *CMSClient) logOperation(op *Ydb_Operations.Operation) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Operation status: %s", op.Status))

	if len(op.Issues) > 0 {
		sb.WriteString(
			fmt.Sprintf("\nIssues:\n%s",
				strings.Join(util.Convert(op.Issues,
					func(issue *Ydb_Issue.IssueMessage) string {
						return fmt.Sprintf("  Severity: %d, code: %d, message: %s", issue.Severity, issue.IssueCode, issue.Message)
					},
				), "\n"),
			))
	}

	c.logger.Debugf("Invocation result:\n%s", sb.String())
}
