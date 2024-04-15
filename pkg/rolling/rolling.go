package rolling

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Discovery"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/auth"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/cms"
	"github.com/ydb-platform/ydbops/pkg/discovery"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type Rolling struct {
	cms       *cms.Client
	discovery *discovery.Client

	factory *client.Factory

	logger    *zap.SugaredLogger
	state     *state
	opts      options.RestartOptions
	restarter restarters.Restarter

	// TODO jorres@: maybe turn this into a local `map`
	// variable in `processActionGroupStates`
	completedActions []*Ydb_Maintenance.ActionUid
}

type state struct {
	nodes                          map[uint32]*Ydb_Maintenance.Node
	tenantNameToNodeIds            map[string][]uint32
	retriesMadeForNode             map[uint32]int
	tenants                        []string
	userSID                        string
	unreportedButFinishedActionIds []string
	restartTaskUID                 string
}

const (
	RestartTaskPrefix = "rolling-restart-"
)

func initAuthToken(
	rootOpts options.RootOptions,
	logger *zap.SugaredLogger,
	factory *client.Factory,
) error {
	switch rootOpts.Auth.Type {
	case options.Static:
		authClient := auth.NewAuthClient(logger, factory)
		staticCreds := rootOpts.Auth.Creds.(*options.AuthStatic)
		user := staticCreds.User
		password := staticCreds.Password
		logger.Debugf("Endpoint: %v", rootOpts.GRPC.Endpoint)
		token, err := authClient.Auth(rootOpts.GRPC, user, password)
		if err != nil {
			return fmt.Errorf("failed to initialize static auth token: %w", err)
		}
		factory.SetAuthToken(token)
	case options.IamToken:
		factory.SetAuthToken(rootOpts.Auth.Creds.(*options.AuthIAMToken).Token)
	case options.IamCreds:
		return fmt.Errorf("TODO: IAM authorization from SA key not implemented yet")
	case options.None:
		return fmt.Errorf("determined credentials to be anonymous. Anonymous credentials are currently unsupported")
	default:
		return fmt.Errorf(
			"internal error: authorization type not recognized after options validation, this should never happen",
		)
	}

	return nil
}

func ExecuteRolling(
	restartOpts options.RestartOptions,
	rootOpts options.RootOptions,
	logger *zap.SugaredLogger,
	restarter restarters.Restarter,
) error {
	factory := client.NewConnectionFactory(rootOpts.Auth, rootOpts.GRPC, restartOpts)

	err := initAuthToken(rootOpts, logger, factory)
	if err != nil {
		return fmt.Errorf("failed to receive an auth token, rolling restart not started: %w", err)
	}

	discoveryClient := discovery.NewDiscoveryClient(logger, factory)
	cmsClient := cms.NewCMSClient(logger, factory)

	r := &Rolling{
		cms:       cmsClient,
		discovery: discoveryClient,
		logger:    logger,
		opts:      restartOpts,
		restarter: restarter,
		factory:   factory,
	}

	if restartOpts.Continue {
		logger.Info("Continue previous rolling restart")
		err = r.DoRestartPrevious()
	} else {
		logger.Info("Start rolling restart")
		err = r.DoRestart()
	}

	if err != nil {
		logger.Errorf("Failed to complete restart: %+v", err)
		return err
	}

	logger.Info("Restart completed successfully")
	return nil
}

func (r *Rolling) DoRestart() error {
	state, err := r.prepareState()
	if err != nil {
		return err
	}
	r.state = state

	if err = r.cleanupOldRollingRestarts(); err != nil {
		return err
	}

	nodeIds, errIds := r.opts.GetNodeIds()
	nodeFQDNs, errFqdns := r.opts.GetNodeFQDNs()
	if errIds != nil && errFqdns != nil {
		return fmt.Errorf(
			"TODO parsing both in id mode and in fqdn mode failed: (%w), (%w)",
			errIds,
			errFqdns,
		)
	}

	nodesToRestart := r.restarter.Filter(
		restarters.FilterNodeParams{
			SelectedTenants: r.opts.TenantList,
			SelectedNodeIds: nodeIds,
			SelectedHosts:   nodeFQDNs,
			StartedTime:     r.opts.StartedTime,
			Version:         r.opts.VersionSpec,
			ExcludeHosts:    r.opts.ExcludeHosts,
		},
		restarters.ClusterNodesInfo{
			TenantToNodeIds: r.state.tenantNameToNodeIds,
			AllNodes:        collections.Values(r.state.nodes),
		},
	)

	taskParams := cms.MaintenanceTaskParams{
		TaskUID:          r.state.restartTaskUID,
		AvailabilityMode: r.opts.GetAvailabilityMode(),
		Duration:         r.opts.GetRestartDuration(),
		Nodes:            nodesToRestart,
	}
	task, err := r.cms.CreateMaintenanceTask(taskParams)
	if err != nil {
		return fmt.Errorf("failed to create maintenance task: %w", err)
	}

	return r.cmsWaitingLoop(task)
}

func (r *Rolling) DoRestartPrevious() error {
	return fmt.Errorf("--continue behavior not implemented yet")
}

func (r *Rolling) cmsWaitingLoop(task cms.MaintenanceTask) error {
	var (
		err          error
		delay        time.Duration
		taskID       = task.GetTaskUid()
		defaultDelay = time.Duration(r.opts.CMSQueryInterval) * time.Second
	)

	r.logger.Infof("Maintenance task %v, processing loop started", taskID)
	for {
		delay = defaultDelay

		if task != nil {
			r.logTask(task)

			if task.GetRetryAfter() != nil {
				retryTime := task.GetRetryAfter().AsTime()
				r.logger.Debugf("Task has retry after attribute: %s", retryTime.Format(time.DateTime))

				if retryDelay := retryTime.Sub(time.Now().UTC()); defaultDelay < retryDelay {
					delay = defaultDelay
				}
			}

			if completed := r.processActionGroupStates(task.GetActionGroupStates()); completed {
				break
			}
		}

		r.logger.Infof("Wait next %s delay", delay)
		time.Sleep(delay)

		r.logger.Infof("Refresh maintenance task with id: %s", taskID)
		task, err = r.cms.RefreshMaintenanceTask(taskID)
		if err != nil {
			r.logger.Warnf("Failed to refresh maintenance task: %+v", err)
		}
	}

	r.logger.Infof("Maintenance task processing loop completed")
	return nil
}

func (r *Rolling) processActionGroupStates(actions []*Ydb_Maintenance.ActionGroupStates) bool {
	r.logger.Debugf("Unfiltered ActionGroupStates: %v", actions)
	performed := collections.FilterBy(actions,
		func(gs *Ydb_Maintenance.ActionGroupStates) bool {
			return gs.ActionStates[0].Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED
		},
	)

	if len(performed) == 0 {
		r.logger.Info("No actions can be taken yet, waiting for CMS to move some actions to PERFORMED...")
		return false
	}

	r.logger.Infof("%d ActionGroupStates moved to PERFORMED, will restart now...", len(performed))

	r.completedActions = []*Ydb_Maintenance.ActionUid{}
	wg := new(sync.WaitGroup)

	rollingStateMutex := new(sync.Mutex)

	for _, gs := range performed {
		var (
			as   = gs.ActionStates[0]
			lock = as.Action.GetLockAction()
			node = r.state.nodes[lock.Scope.GetNodeId()]
		)

		if collections.Contains(r.state.unreportedButFinishedActionIds, as.ActionUid.ActionId) {
			r.completedActions = append(r.completedActions, as.ActionUid)
			r.logger.Debugf(
				"Node id %v already restarted, but CompleteAction failed on last iteration, "+
					"so CMS does not know it is complete yet.",
				node.NodeId,
			)
			continue
		}

		r.logger.Debugf("Drain node with id: %d", node.NodeId)
		wg.Add(1)

		go func() {
			defer wg.Done()

			// TODO: drain node, but public draining api is not available yet
			r.logger.Warn("DRAINING NOT IMPLEMENTED YET")

			r.logger.Debugf("Restart node with id: %d", node.NodeId)
			if err := r.restarter.RestartNode(node); err != nil {
				rollingStateMutex.Lock()
				retriesUntilNow := r.state.retriesMadeForNode[node.NodeId]
				r.state.retriesMadeForNode[node.NodeId]++
				rollingStateMutex.Unlock()

				r.logger.Warnf(
					"Failed to restart node with id: %d, attempt number %v, because of: %s",
					node.NodeId,
					retriesUntilNow,
					err.Error(),
				)

				if retriesUntilNow+1 == r.opts.RestartRetryNumber {
					r.atomicRememberComplete(rollingStateMutex, as.ActionUid)
					r.logger.Warnf("Failed to retry node %v specified number of times %v", node.NodeId, r.opts.RestartRetryNumber)
				}
			} else {
				r.atomicRememberComplete(rollingStateMutex, as.ActionUid)
				r.logger.Debugf("Successfully restarted node with id: %d", node.NodeId)
			}
		}()
	}

	wg.Wait()

	result, err := r.cms.CompleteAction(r.completedActions)
	if err != nil {
		r.logger.Warnf("Failed to complete action: %+v", err)
		return false
	}
	r.logCompleteResult(result)
	r.state.unreportedButFinishedActionIds = []string{}

	// completed when all actions marked as completed
	return len(actions) == len(result.ActionStatuses)
}

func (r *Rolling) atomicRememberComplete(m *sync.Mutex, actionUID *Ydb_Maintenance.ActionUid) {
	m.Lock()
	defer m.Unlock()

	r.state.unreportedButFinishedActionIds = append(r.state.unreportedButFinishedActionIds, actionUID.ActionId)
	r.completedActions = append(r.completedActions, actionUID)
}

func (r *Rolling) prepareState() (*state, error) {
	tenants, err := r.cms.Tenants()
	if err != nil {
		return nil, fmt.Errorf("failed to list available tenants: %w", err)
	}

	tenantNameToNodeIds := make(map[string][]uint32)

	for _, tenant := range r.opts.TenantList {
		if collections.Contains(r.opts.TenantList, tenant) {
			return nil, fmt.Errorf("tenant %s is not found in tenant list of this cluster", tenant)
		}

		endpoints, err := r.discovery.ListEndpoints(tenant)
		if err != nil {
			return nil, fmt.Errorf("failed to get the list of endpoints for the tenant %s", tenant)
		}
		tenantNameToNodeIds[tenant] = collections.Convert(endpoints, func(endpoint *Ydb_Discovery.EndpointInfo) uint32 {
			return endpoint.NodeId
		})
	}

	nodes, err := r.cms.Nodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list available nodes: %w", err)
	}

	userSID, err := r.discovery.WhoAmI()
	if err != nil {
		return nil, fmt.Errorf("failed to determine the user SID: %w", err)
	}

	return &state{
		tenantNameToNodeIds:            tenantNameToNodeIds,
		tenants:                        tenants,
		userSID:                        userSID,
		nodes:                          collections.ToMap(nodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
		retriesMadeForNode:             make(map[uint32]int),
		unreportedButFinishedActionIds: []string{},
		restartTaskUID:                 RestartTaskPrefix + uuid.New().String(),
	}, nil
}

func (r *Rolling) cleanupOldRollingRestarts() error {
	r.logger.Debugf("Will cleanup all previous maintenance tasks...")

	previousTasks, err := r.cms.MaintenanceTasks(r.state.userSID)
	if err != nil {
		return fmt.Errorf("failed to list maintenance tasks with user id %v: %w", r.state.userSID, err)
	}

	for _, previousTaskUID := range previousTasks {
		_, err := r.cms.DropMaintenanceTask(previousTaskUID)
		if err != nil {
			return fmt.Errorf("failed to drop maintenance task: %w", err)
		}
	}
	return nil
}

func (r *Rolling) logTask(task cms.MaintenanceTask) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Uid: %s\n", task.GetTaskUid()))

	if task.GetRetryAfter() != nil {
		sb.WriteString(fmt.Sprintf("Retry after: %s\n", task.GetRetryAfter().AsTime().Format(time.DateTime)))
	}

	for _, gs := range task.GetActionGroupStates() {
		as := gs.ActionStates[0]
		sb.WriteString(fmt.Sprintf("  Lock on node %d ", as.Action.GetLockAction().Scope.GetNodeId()))
		if as.Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED {
			sb.WriteString(fmt.Sprintf("PERFORMED, until: %s", as.Deadline.AsTime().Format(time.DateTime)))
		} else {
			sb.WriteString(fmt.Sprintf("PENDING, %s", as.GetReason().String()))
		}
		sb.WriteString("\n")
	}
	r.logger.Debugf("Maintenance task result:\n%s", sb.String())
}

func (r *Rolling) logCompleteResult(result *Ydb_Maintenance.ManageActionResult) {
	if result == nil {
		return
	}

	sb := strings.Builder{}

	for _, status := range result.ActionStatuses {
		sb.WriteString(fmt.Sprintf("  Action: %s, status: %s", status.ActionUid, status.Status))
	}

	r.logger.Debugf("Manage action result:\n%s", sb.String())
}
