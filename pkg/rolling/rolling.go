package rolling

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydb-ops/internal/collections"
	"github.com/ydb-platform/ydb-ops/pkg/auth"
	"github.com/ydb-platform/ydb-ops/pkg/client"
	"github.com/ydb-platform/ydb-ops/pkg/cms"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"

	"github.com/ydb-platform/ydb-ops/pkg/discovery"
	"github.com/ydb-platform/ydb-ops/pkg/options"
)

type Rolling struct {
	cms       *cms.CMSClient
	discovery *discovery.DiscoveryClient

	factory *client.Factory

	logger    *zap.SugaredLogger
	state     *state
	opts      *options.RestartOptions
	restarter restarters.Restarter
}

type state struct {
	nodes                          map[uint32]*Ydb_Maintenance.Node
	tenants                        []string
	userSID                        string
	unreportedButFinishedActionIds []string
	restartTaskUID                 string
}

const (
	RestartTaskPrefix = "rolling-restart-"
)

func initAuthToken(
	rootOpts *options.RootOptions,
	logger *zap.SugaredLogger,
	factory *client.Factory,
) error {

	switch rootOpts.Auth.Type {
	case options.Static:
		authClient := auth.NewAuthClient(logger, factory)
		staticCreds := rootOpts.Auth.Creds.(*options.AuthStatic)
		user := staticCreds.User
		password := staticCreds.Password
		token, err := authClient.Auth(rootOpts.GRPC, user, password)
		if err != nil {
			return fmt.Errorf("Failed to initialize static auth token: %w", err)
		}
		factory.SetAuthToken(token)
	case options.IamToken:
		factory.SetAuthToken(rootOpts.Auth.Creds.(*options.AuthIAMToken).Token)
	case options.IamCreds:
		return fmt.Errorf("TODO: IAM authorization from SA key not implemented yet")
	case options.None:
		factory.SetAuthToken("")
	default:
		return fmt.Errorf("Internal error: authorization type not recognized after options validation, this should never happen")
	}

	return nil
}

func PrepareRolling(
	restartOpts *options.RestartOptions,
	rootOpts *options.RootOptions,
	logger *zap.SugaredLogger,
	restarter restarters.Restarter,
) {
	factory := client.NewConnectionFactory(rootOpts.Auth, rootOpts.GRPC)

	logger.Debugf("rootOpts.Auth.Type %v", rootOpts.Auth.Type)
	err := initAuthToken(rootOpts, logger, factory)
	if err != nil {
		logger.Errorf("Failed to begin restart loop: %+v", err)
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
	} else {
		logger.Info("Restart completed successfully")
	}
}

func (r *Rolling) DoRestart() error {
	state, err := r.prepareState()
	if err != nil {
		return err
	}
	r.state = state

	if err := r.cleanupOldRollingRestarts(); err != nil {
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
		r.logger,
		restarters.FilterNodeParams{
			SelectedTenants:   r.opts.Tenants,
			SelectedNodeIds:   nodeIds,
			SelectedHostFQDNs: nodeFQDNs,
		},
		restarters.ClusterNodesInfo{
			AllTenants: r.state.tenants,
			AllNodes:   collections.Values(r.state.nodes),
		},
	)

	taskParams := cms.MaintenanceTaskParams{
		TaskUID:          r.state.restartTaskUID,
		AvailAbilityMode: r.opts.GetAvailabilityMode(),
		Duration:         r.opts.GetRestartDuration(),
		Nodes:            nodesToRestart,
	}
	task, err := r.cms.CreateMaintenanceTask(taskParams)
	if err != nil {
		return fmt.Errorf("failed to create maintenance task: %+v", err)
	}

	r.logger.Infof("Maintenance task id: %s", task.GetTaskUid())
	return r.cmsWaitingLoop(task)
}

func (r *Rolling) DoRestartPrevious() error {
	return fmt.Errorf("--continue behaviour not implemented yet.")
}

func (r *Rolling) cmsWaitingLoop(task cms.MaintenanceTask) error {
	const (
		defaultDelay = time.Second * 10
	)

	var (
		err    error
		delay  time.Duration
		taskId = task.GetTaskUid()
	)

	r.logger.Infof("Maintenance task processing loop started")
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

			r.logger.Info("Processing task action group states")
			r.logger.Debug(r.state.unreportedButFinishedActionIds)
			if completed := r.processActionGroupStates(task.GetActionGroupStates()); completed {
				break
			}
		}

		r.logger.Infof("Wait next %s delay", delay)
		time.Sleep(delay)

		r.logger.Infof("Refresh maintenance task with id: %s", taskId)
		task, err = r.cms.RefreshMaintenanceTask(taskId)
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

	r.logger.Infof("Perform next %d ActionGroupStates", len(performed))
	actionsCompletedThisStep := []*Ydb_Maintenance.ActionUid{}
	for _, gs := range performed {
		var (
			as   = gs.ActionStates[0]
			lock = as.Action.GetLockAction()
			node = r.state.nodes[lock.Scope.GetNodeId()]
		)

		if collections.Contains(r.state.unreportedButFinishedActionIds, as.ActionUid.ActionId) {
			actionsCompletedThisStep = append(actionsCompletedThisStep, as.ActionUid)
			r.logger.Debugf(
				"Node id %v already restarted, but CompleteAction failed on last iteration, so CMS does not know it is complete yet.",
				node.NodeId,
			)
			continue
		}

		r.logger.Debugf("Drain node with id: %d", node.NodeId)

		r.logger.Warn("DRAINING NOT IMPLEMENTED YET")
		// TODO: drain node, but public draining api is not available yet

		r.logger.Debugf("Restart node with id: %d", node.NodeId)
		if err := r.restarter.RestartNode(r.logger, node); err != nil {
			r.logger.Warnf("Failed to restart node with id: %d, because of: %w", node.NodeId, err)
		} else {
			r.state.unreportedButFinishedActionIds = append(r.state.unreportedButFinishedActionIds, as.ActionUid.ActionId)
			actionsCompletedThisStep = append(actionsCompletedThisStep, as.ActionUid)
			r.logger.Debugf("Successfully restarted node with id: %d", node.NodeId)
		}
	}

	result, err := r.cms.CompleteAction(actionsCompletedThisStep)
	if err != nil {
		r.logger.Warnf("Failed to complete action: %+v", err)
		return false
	}
	r.logCompleteResult(result)
	r.state.unreportedButFinishedActionIds = []string{}

	// completed when all actions marked as completed
	return len(actions) == len(result.ActionStatuses)
}

func (r *Rolling) prepareState() (*state, error) {
	tenants, err := r.cms.Tenants()
	if err != nil {
		return nil, fmt.Errorf("failed to list available tenants: %+v", err)
	}

	nodes, err := r.cms.Nodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list available nodes: %+v", err)
	}

	userSID, err := r.discovery.WhoAmI()
	if err != nil {
		return nil, fmt.Errorf("whoami failed: %+v", err)
	}

	return &state{
		tenants:                        tenants,
		userSID:                        userSID,
		nodes:                          collections.ToMap(nodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
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
