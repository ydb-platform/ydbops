package rolling

import (
	"fmt"
	"strings"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/cms"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"

	"github.com/ydb-platform/ydb-ops/pkg/options"
)

type Rolling struct {
	logger    *zap.SugaredLogger
	cms       *cms.CMSClient
	state     *state
	opts      *options.RestartOptions
	restarter restarters.RestarterInterface
}

type state struct {
	nodes   map[uint32]*Ydb_Maintenance.Node
	tenants []string
}

const (
	RestartTaskPrefix  = "rolling_restart"
	RestartTaskUid     = RestartTaskPrefix + "_001"
	RollingRestartUser = "rolling-restart"
)

func PrepareRolling(restartOpts *options.RestartOptions, rootOpts *options.RootOptions, lf *zap.Logger, restarter restarters.RestarterInterface) {
	var err error
	logger := lf.Sugar()

	cmsClient := cms.NewCMSClient(logger,
		cms.NewConnectionFactory(
			*restartOpts.CMS, // TODO gain deep understanding, why dereferencing is necessary
			*rootOpts,
			RollingRestartUser,
		),
	)

	r := &Rolling{
		cms:       cmsClient,
		logger:    logger,
		opts:      restartOpts,
		restarter: restarter,
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

	nodeIds, _ := r.opts.GetNodeIds()
	nodesToRestart := r.restarter.Filter(
		restarters.FilterNodeParams{
			AllTenants:      r.state.tenants,
			AllNodes:        util.Values(r.state.nodes),
			SelectedTenants: r.opts.Tenants,
			SelectedNodeIds: nodeIds,
		},
	)
	taskParams := cms.MaintenanceTaskParams{
		TaskUid:          RestartTaskUid,
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
	state, err := r.prepareState()
	if err != nil {
		return err
	}
	r.state = state

	result, err := r.cms.GetMaintenanceTask(RestartTaskUid)
	if err != nil {
		return fmt.Errorf("failed to get maintenance task with id: %s, err: %+v", RestartTaskUid, err)
	}

	return r.cmsWaitingLoop(result)
}

func (r *Rolling) cmsWaitingLoop(task cms.MaintenanceTask) error {
	const (
		defaultDelay = time.Second * 30
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
	performed := util.FilterBy(actions,
		func(gs *Ydb_Maintenance.ActionGroupStates) bool {
			return gs.ActionStates[0].Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED
		},
	)

	if len(performed) == 0 {
		r.logger.Info("No unprocessed ActionGroupStates moved to Performed state yet.")
		return false
	}

	ids := make([]*Ydb_Maintenance.ActionUid, 0, len(performed))

	r.logger.Infof("Perform next %d ActionGroupStates", len(performed))
	for _, gs := range performed {
		var (
			as   = gs.ActionStates[0]
			lock = as.Action.GetLockAction()
			node = r.state.nodes[lock.Scope.GetNodeId()]
		)

		r.logger.Debugf("Drain node with id: %d", node.NodeId)
		// TODO: drain node

		r.logger.Debugf("Restart node with id: %d", node.NodeId)
		if err := r.restarter.RestartNode(node); err != nil {
			r.logger.Warnf("Failed to restart node with id: %d", node.NodeId)
		}

		ids = append(ids, as.ActionUid)
	}

	result, err := r.cms.CompleteAction(ids)
	if err != nil {
		r.logger.Warnf("Failed to complete action: %+v", err)
		return false
	}
	r.logCompleteResult(result)

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

	return &state{
		tenants: tenants,
		nodes:   util.ToMap(nodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
	}, nil
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
			sb.WriteString("PENDING")
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
