package cms

import (
	"context"
	"fmt"

	"github.com/ydb-platform/ydb-go-genproto/Ydb_Cms_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/Ydb_Maintenance_V1"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Cms"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/client/auth/credentials"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	defaultRetryCount = 5
)

type CMS interface {
	Tenants() ([]string, error)
	Nodes() ([]*Ydb_Maintenance.Node, error)
}

type Client interface {
	CMS
	Maintenance

	Close() error
}

type defaultCMSClient struct {
	logger              *zap.SugaredLogger
	connectionsFactory  connectionsfactory.Factory
	credentialsProvider credentials.Provider
}

func NewCMSClient(
	connectionsFactory connectionsfactory.Factory,
	logger *zap.SugaredLogger,
	cp credentials.Provider,
) Client {
	return &defaultCMSClient{
		logger:              logger,
		connectionsFactory:  connectionsFactory,
		credentialsProvider: cp,
	}
}

func (c *defaultCMSClient) Tenants() ([]string, error) {
	result := Ydb_Cms.ListDatabasesResult{}
	c.logger.Debug("Invoke ListDatabases method")
	_, err := c.executeCMSOperation(&result, func(ctx context.Context, cl Ydb_Cms_V1.CmsServiceClient) (client.OperationResponse, error) {
		return cl.ListDatabases(ctx, &Ydb_Cms.ListDatabasesRequest{OperationParams: c.connectionsFactory.OperationParams()})
	})
	if err != nil {
		return nil, err
	}

	s := collections.SortBy(result.Paths,
		func(l string, r string) bool {
			return l < r
		},
	)
	return s, nil
}

func (c *defaultCMSClient) Nodes() ([]*Ydb_Maintenance.Node, error) {
	result := Ydb_Maintenance.ListClusterNodesResult{}
	c.logger.Debug("Invoke ListClusterNodes method")
	_, err := c.executeMaintenanceOperation(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.ListClusterNodes(ctx, &Ydb_Maintenance.ListClusterNodesRequest{
				OperationParams: c.connectionsFactory.OperationParams(),
			})
		},
	)
	if err != nil {
		return nil, err
	}

	nodes := collections.SortBy(result.Nodes,
		func(l *Ydb_Maintenance.Node, r *Ydb_Maintenance.Node) bool {
			return l.NodeId < r.NodeId
		},
	)

	return nodes, nil
}

func (c *defaultCMSClient) MaintenanceTasks(userSID string) ([]MaintenanceTask, error) {
	result := Ydb_Maintenance.ListMaintenanceTasksResult{}
	c.logger.Debug("Invoke ListMaintenanceTasks method")
	_, err := c.executeMaintenanceOperation(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.ListMaintenanceTasks(ctx,
				&Ydb_Maintenance.ListMaintenanceTasksRequest{
					OperationParams: c.connectionsFactory.OperationParams(),
					User:            &userSID,
				},
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return c.queryEachTaskForActions(result.TasksUids)
}

func (c *defaultCMSClient) GetMaintenanceTask(taskID string) (MaintenanceTask, error) {
	result := Ydb_Maintenance.GetMaintenanceTaskResult{}
	c.logger.Debug("Invoke GetMaintenanceTask method")
	_, err := c.executeMaintenanceOperation(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.GetMaintenanceTask(ctx, &Ydb_Maintenance.GetMaintenanceTaskRequest{
				OperationParams: c.connectionsFactory.OperationParams(),
				TaskUid:         taskID,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return &maintenanceTaskResult{
		TaskUID:           taskID,
		ActionGroupStates: result.ActionGroupStates,
	}, nil
}

func wrapSingleScopeInActionGroup(
	scope *Ydb_Maintenance.ActionScope,
	duration *durationpb.Duration,
) *Ydb_Maintenance.ActionGroup {
	return &Ydb_Maintenance.ActionGroup{
		Actions: []*Ydb_Maintenance.Action{
			{
				Action: &Ydb_Maintenance.Action_LockAction{
					LockAction: &Ydb_Maintenance.LockAction{
						Scope:    scope,
						Duration: duration,
					},
				},
			},
		},
	}
}

func actionGroupsFromNodes(params MaintenanceTaskParams) []*Ydb_Maintenance.ActionGroup {
	ags := make([]*Ydb_Maintenance.ActionGroup, 0, len(params.Nodes))

	for _, node := range params.Nodes {
		scope := &Ydb_Maintenance.ActionScope{
			Scope: &Ydb_Maintenance.ActionScope_NodeId{
				NodeId: node.NodeId,
			},
		}

		ags = append(ags, wrapSingleScopeInActionGroup(scope, params.Duration))
	}

	return ags
}

func actionGroupsFromHosts(params MaintenanceTaskParams) []*Ydb_Maintenance.ActionGroup {
	ags := make([]*Ydb_Maintenance.ActionGroup, 0, len(params.Hosts))

	for _, hostFQDN := range params.Hosts {
		scope := &Ydb_Maintenance.ActionScope{
			Scope: &Ydb_Maintenance.ActionScope_Host{
				Host: hostFQDN,
			},
		}

		ags = append(ags, wrapSingleScopeInActionGroup(scope, params.Duration))
	}

	return ags
}

func (c *defaultCMSClient) CreateMaintenanceTask(params MaintenanceTaskParams) (MaintenanceTask, error) {
	request := &Ydb_Maintenance.CreateMaintenanceTaskRequest{
		OperationParams: c.connectionsFactory.OperationParams(),
		TaskOptions: &Ydb_Maintenance.MaintenanceTaskOptions{
			TaskUid:          params.TaskUID,
			AvailabilityMode: params.AvailabilityMode,
			Description:      "Rolling restart maintenance task",
		},
	}

	fmt.Println(params.Duration)
	if params.ScopeType == NodeScope {
		request.ActionGroups = actionGroupsFromNodes(params)
	} else { // HostScope
		request.ActionGroups = actionGroupsFromHosts(params)
	}

	result := &Ydb_Maintenance.MaintenanceTaskResult{}
	c.logger.Debug("Invoke CreateMaintenanceTask method")
	_, err := c.executeMaintenanceOperation(result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.CreateMaintenanceTask(ctx, request)
		},
	)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *defaultCMSClient) RefreshMaintenanceTask(taskID string) (MaintenanceTask, error) {
	result := Ydb_Maintenance.MaintenanceTaskResult{}
	c.logger.Debug("Invoke RefreshMaintenanceTask method")
	_, err := c.executeMaintenanceOperation(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.RefreshMaintenanceTask(ctx, &Ydb_Maintenance.RefreshMaintenanceTaskRequest{
				OperationParams: c.connectionsFactory.OperationParams(),
				TaskUid:         taskID,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *defaultCMSClient) DropMaintenanceTask(taskID string) (string, error) {
	c.logger.Debug("Invoke DropMaintenanceTask method")
	op, err := c.executeMaintenanceOperation(nil,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.DropMaintenanceTask(ctx, &Ydb_Maintenance.DropMaintenanceTaskRequest{
				OperationParams: c.connectionsFactory.OperationParams(),
				TaskUid:         taskID,
			})
		},
	)
	if err != nil {
		return "", err
	}

	return op.Status.String(), nil
}

func (c *defaultCMSClient) CompleteAction(actionIds []*Ydb_Maintenance.ActionUid) (*Ydb_Maintenance.ManageActionResult, error) {
	result := Ydb_Maintenance.ManageActionResult{}
	c.logger.Debug("Invoke CompleteAction method")
	_, err := c.executeMaintenanceOperation(&result,
		func(ctx context.Context, cl Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error) {
			return cl.CompleteAction(ctx, &Ydb_Maintenance.CompleteActionRequest{
				OperationParams: c.connectionsFactory.OperationParams(),
				ActionUids:      actionIds,
			})
		},
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *defaultCMSClient) executeMaintenanceOperation(
	out proto.Message,
	method func(context.Context, Ydb_Maintenance_V1.MaintenanceServiceClient) (client.OperationResponse, error),
) (*Ydb_Operations.Operation, error) {
	ctx, cancel := c.credentialsProvider.ContextWithAuth(context.TODO())
	defer cancel()

	op, err := utils.WrapWithRetries(defaultRetryCount, func() (*Ydb_Operations.Operation, error) {
		cc, err := c.connectionsFactory.Create()
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = cc.Close()
		}()

		cl := Ydb_Maintenance_V1.NewMaintenanceServiceClient(cc)
		r, err := method(ctx, cl)
		if err != nil {
			c.logger.Debugf("Invocation error: %+v", err)
			return nil, err
		}
		op := r.GetOperation()
		utils.LogOperation(c.logger, op)
		return op, nil
	})
	if err != nil {
		return nil, err
	}

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

func (c *defaultCMSClient) executeCMSOperation(
	out proto.Message,
	method func(context.Context, Ydb_Cms_V1.CmsServiceClient) (client.OperationResponse, error),
) (*Ydb_Operations.Operation, error) {
	ctx, cancel := c.credentialsProvider.ContextWithAuth(context.TODO())
	defer cancel()

	op, err := utils.WrapWithRetries(defaultRetryCount, func() (*Ydb_Operations.Operation, error) {
		cc, err := c.connectionsFactory.Create()
		if err != nil {
			return nil, err
		}

		cl := Ydb_Cms_V1.NewCmsServiceClient(cc)
		r, err := method(ctx, cl)
		if err != nil {
			c.logger.Debugf("Invocation error: %+v", err)
			return nil, err
		}
		op := r.GetOperation()
		utils.LogOperation(c.logger, op)
		return op, nil
	})
	if err != nil {
		return nil, err
	}

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

func (c *defaultCMSClient) Close() error {
	return nil
}
