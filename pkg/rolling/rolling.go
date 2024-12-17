package rolling

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/client/discovery"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

type Rolling struct {
	cms       cms.Client
	discovery discovery.Client

	logger    *zap.SugaredLogger
	state     *state
	opts      *RestartOptions
	restarter restarters.Restarter

	// TODO jorres@: maybe turn this into a local `map`
	// variable in `processActionGroupStates`
	completedActions []*Ydb_Maintenance.ActionUid
	mu               sync.RWMutex
}

type MajorToMinors map[int]map[int]bool

type state struct {
	knownVersions                  MajorToMinors
	nodes                          map[uint32]*Ydb_Maintenance.Node
	inactiveNodes                  map[uint32]*Ydb_Maintenance.Node
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

type Executer interface {
	Execute() error
}

type executer struct {
	cmsClient       cms.Client
	discoveryClient discovery.Client
	opts            *RestartOptions
	logger          *zap.SugaredLogger
	restarter       restarters.Restarter
}

func NewExecuter(
	opts *RestartOptions,
	logger *zap.SugaredLogger,
	cmsClient cms.Client,
	discoveryClient discovery.Client,
	rst restarters.Restarter,
) Executer {
	return &executer{
		cmsClient:       cmsClient,
		discoveryClient: discoveryClient,
		logger:          logger,
		restarter:       rst,
		opts:            opts, // TODO(shmel1k@): create own options
	}
}

func (e *executer) Execute() error {
	r := &Rolling{
		cms:       e.cmsClient,
		discovery: e.discoveryClient,
		logger:    e.logger,
		opts:      e.opts,
		restarter: e.restarter,
	}

	var err error
	if e.opts.Continue {
		e.logger.Info("Continue previous rolling restart")
		err = r.DoRestartPrevious()
	} else {
		e.logger.Info("Start rolling restart")
		err = r.DoRestart()
	}

	if err != nil {
		e.logger.Errorf("Failed to complete restart: %+v", err)
		return err
	}

	e.logger.Info("Restart completed successfully")
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

	nodeIds, errIds := utils.GetNodeIds(r.opts.Hosts)
	nodeFQDNs, errFqdns := utils.GetNodeFQDNs(r.opts.Hosts)
	if errIds != nil && errFqdns != nil {
		return fmt.Errorf(
			"TODO parsing both in id mode and in fqdn mode failed: (%w), (%w)",
			errIds,
			errFqdns,
		)
	}

	nodesToRestart := r.restarter.Filter(
		restarters.FilterNodeParams{
			SelectedTenants:     r.opts.TenantList,
			SelectedNodeIds:     nodeIds,
			SelectedHosts:       nodeFQDNs,
			SelectedDatacenters: r.opts.Datacenters,
			StartedTime:         r.opts.StartedTime,
			Version:             r.opts.VersionSpec,
			ExcludeHosts:        r.opts.ExcludeHosts,
			MaxStaticNodeID:     uint32(r.opts.MaxStaticNodeID),
		},
		restarters.ClusterNodesInfo{
			TenantToNodeIds: r.state.tenantNameToNodeIds,
			AllNodes:        collections.Values(r.state.nodes),
		},
	)

	excludedNodes := 0
	for _, node := range nodesToRestart {
		if _, present := r.state.inactiveNodes[node.NodeId]; present {
			excludedNodes++
			r.logger.Warn(
				"the node with nodeId: %v and host: %s is currently down and will be excluded from restart",
				node.Host,
				node.NodeId,
			)
		}
	}

	if len(nodesToRestart)-excludedNodes == 0 {
		r.logger.Warn("There are no nodes that satisfy the specified filters")
		return nil
	}

	taskParams := cms.MaintenanceTaskParams{
		TaskUID:          r.state.restartTaskUID,
		AvailabilityMode: r.opts.GetAvailabilityMode(),
		Duration:         r.opts.GetRestartDuration(),
		ScopeType:        cms.NodeScope,
		Nodes:            nodesToRestart,
	}

	task, err := r.cms.CreateMaintenanceTask(taskParams)
	if err != nil {
		return fmt.Errorf("failed to create maintenance task: %w", err)
	}

	return r.cmsWaitingLoop(task, len(nodesToRestart))
}

func (r *Rolling) DoRestartPrevious() error {
	return fmt.Errorf("--continue behavior not implemented yet")
}

func (r *Rolling) cmsWaitingLoop(task cms.MaintenanceTask, totalNodes int) error {
	var (
		err          error
		delay        time.Duration
		taskID       = task.GetTaskUid()
		defaultDelay = time.Duration(r.opts.CMSQueryInterval) * time.Second
	)

	restartedNodes := 0

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

			if completed := r.processActionGroupStates(task.GetActionGroupStates(), &restartedNodes); completed {
				break
			}
		}

		r.logger.Infof("Wait next %s delay. Total node progress: %v out of %v", delay, restartedNodes, totalNodes)
		time.Sleep(delay)

		r.logger.Infof("Refresh maintenance task with id: %s", taskID)
		task, err = r.cms.RefreshMaintenanceTask(taskID)
		if err != nil {
			r.logger.Warnf("Failed to refresh maintenance task: %+v", err)
			continue
		}

		// NOTE: compatibility check will not fire if rolling restart just
		// finished restarting last nodes. It will only fire when some
		// nodes still remain and we are waiting for CMS anyway.

		// But since 99 out of 100 times we want to rolling-restart more than
		// 1 node, this check will fire at least once and it would be enough.

		// In other words, if the whole cluster has restarted without
		// compatibility issues, we don't want to force the user to wait extra
		// tens of seconds after last iteration to simply check compatibility
		// issues once more. We better exit quickly.
		if !r.opts.SuppressCompatibilityCheck {
			incompatible := r.tryDetectCompatibilityIssues()

			// if error is retryExceeded, just keep trying - maybe you have been asking CMS
			// from a node that has just been restarted, and it's okay.
			if incompatible != nil && !errors.Is(incompatible, &utils.RetryExceededError{}) {
				return incompatible
			}
		}
	}

	r.logger.Infof("Maintenance task processing loop completed")
	return nil
}

func (r *Rolling) handleRestartStatus(
	statuses <-chan restartStatus,
	restartedNodes *int,
	expectedRestarts int,
) {
	for i := 0; i < expectedRestarts; i++ {
		st := <-statuses
		if st.err == nil {
			r.atomicRememberComplete(st.as.GetActionUid(), restartedNodes)
			continue
		}

		retriesUntilNow := r.state.retriesMadeForNode[st.nodeID]
		r.state.retriesMadeForNode[st.nodeID]++

		r.logger.Warnf(
			"Failed to restart node with id: %d, attempt number %v, because of: %s",
			st.nodeID,
			retriesUntilNow,
			st.err.Error(),
		)

		if retriesUntilNow+1 == r.opts.RestartRetryNumber {
			r.atomicRememberComplete(st.as.GetActionUid(), restartedNodes)
			r.logger.Warnf("Failed to retry node %v specified number of times (%v)", st.nodeID, r.opts.RestartRetryNumber)
		}
	}
}

func (r *Rolling) processActionGroupStates(actions []*Ydb_Maintenance.ActionGroupStates, restartedNodes *int) bool {
	r.logger.Debugf("Unfiltered ActionGroupStates: %v", actions)

	var actionStatesBuf bytes.Buffer
	performed := collections.FilterBy(actions,
		func(gs *Ydb_Maintenance.ActionGroupStates) bool {
			st := gs.ActionStates[0]
			if st.Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED {
				return true
			}

			details := st.GetReasonDetails()
			if details != "" {
				actionStatesBuf.WriteString(st.GetReasonDetails())
				actionStatesBuf.WriteString("\n")
			}
			return false
		},
	)

	if len(performed) == 0 {
		msg := actionStatesBuf.String()
		if len(msg) == 0 {
			msg = "<CMS does not support reporting details in this YDB version yet>"
		}

		r.logger.Info("No actions can be taken yet, CMS didn't move any actions to PERFORMED because of: ", msg)
		return false
	}

	r.logger.Infof("%d ActionGroupStates moved to PERFORMED, will restart now...", len(performed))

	r.completedActions = []*Ydb_Maintenance.ActionUid{}

	statusCh := make(chan restartStatus, r.opts.NodesInflight)
	restartHandler := newRestartHandler(
		r.logger,
		r.restarter,
		r.opts.NodesInflight,
		r.opts.DelayBetweenRestarts,
		r.state.nodes,
		statusCh,
	)
	restartHandler.run()
	done := make(chan struct{})

	expectedRestarts := 0
	for _, gs := range performed {
		var (
			as   = gs.ActionStates[0]
			lock = as.Action.GetLockAction()
			node = r.state.nodes[lock.Scope.GetNodeId()]
		)
		if r.atomicHasActionInUnreported(as.GetActionUid().GetActionId()) {
			r.mu.Lock()
			r.completedActions = append(r.completedActions, as.ActionUid)
			r.mu.Unlock()

			r.logger.Debugf(
				"Node id %v already restarted, but CompleteAction failed on last iteration, "+
					"so CMS does not know it is complete yet.",
				node.NodeId,
			)
			continue
		}
		expectedRestarts++
	}

	go func() {
		r.handleRestartStatus(statusCh, restartedNodes, expectedRestarts)
		close(done)
	}()

	for _, gs := range performed {
		as := gs.ActionStates[0]
		if r.atomicHasActionInUnreported(as.ActionUid.GetActionId()) {
			continue
		}
		restartHandler.push(gs)
	}

	<-done
	restartHandler.stop()
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

func (r *Rolling) atomicHasActionInUnreported(actionID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return collections.Contains(r.state.unreportedButFinishedActionIds, actionID)
}

func (r *Rolling) atomicRememberComplete(actionUID *Ydb_Maintenance.ActionUid, restartedNodes *int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state.unreportedButFinishedActionIds = append(r.state.unreportedButFinishedActionIds, actionUID.ActionId)
	r.completedActions = append(r.completedActions, actionUID)
	(*restartedNodes)++
}

func (r *Rolling) prepareState() (*state, error) {
	nodes, err := r.cms.Nodes()

	inactiveNodes := collections.FilterBy(nodes, func(node *Ydb_Maintenance.Node) bool {
		return node.GetState() != Ydb_Maintenance.ItemState_ITEM_STATE_UP
	})

	activeNodes := collections.FilterBy(nodes, func(node *Ydb_Maintenance.Node) bool {
		return node.GetState() == Ydb_Maintenance.ItemState_ITEM_STATE_UP
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list available nodes: %w", err)
	}

	tenants, err := r.cms.Tenants()
	if err != nil {
		return nil, fmt.Errorf("failed to list available tenants: %w", err)
	}
	for _, tenant := range r.opts.TenantList {
		if !collections.Contains(tenants, tenant) {
			return nil, fmt.Errorf("tenant %s is not found in tenant list of this cluster", tenant)
		}
	}

	userSID, err := r.discovery.WhoAmI()
	if err != nil {
		return nil, fmt.Errorf("failed to determine the user SID: %w", err)
	}

	return &state{
		knownVersions:                  make(MajorToMinors),
		tenantNameToNodeIds:            utils.PopulateTenantToNodesMapping(activeNodes),
		tenants:                        tenants,
		userSID:                        userSID,
		nodes:                          collections.ToMap(activeNodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
		inactiveNodes:                  collections.ToMap(inactiveNodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
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
		_, err := r.cms.DropMaintenanceTask(previousTaskUID.GetTaskUid())
		if err != nil {
			return fmt.Errorf("failed to drop maintenance task: %w", err)
		}
	}
	return nil
}

func findLowHigh(minors map[int]bool) (low, high int) {
	low = math.MaxInt
	high = math.MinInt

	for minor := range minors {
		if minor < low {
			low = minor
		}
		if minor > high {
			high = minor
		}
	}

	return low, high
}

func checkWithinOneMajor(major int, minors map[int]bool, message *strings.Builder) {
	low, high := findLowHigh(minors)
	if high-low > 1 {
		fmt.Fprintf(message, "%v-%v with %v-%v, minors too far\n", major, high, major, low)
	}
}

func checkWithPreviousMajors(curMajor int, knownVersions MajorToMinors, message *strings.Builder) {
	for prevMajor, prevMinors := range knownVersions {
		if prevMajor >= curMajor {
			continue
		}

		prevLow, prevHigh := findLowHigh(prevMinors)
		if prevLow != prevHigh {
			fmt.Fprintf(message, "%v major is incompatible with %v-%v\n", curMajor, prevMajor, prevLow)
		}
	}
}

func (r *Rolling) tryDetectCompatibilityIssues() error {
	nodes, err := r.cms.Nodes()
	if err != nil {
		return fmt.Errorf("failed to fetch nodes while checking for compatibility issues: %w", err)
	}

	unknownVersionNodes := 0
	for _, node := range nodes {
		major, minor, _, err := utils.ParseMajorMinorPatchFromVersion(node.Version)
		if err == nil {
			if _, exists := r.state.knownVersions[major]; !exists {
				r.state.knownVersions[major] = make(map[int]bool)
			}
			r.state.knownVersions[major][minor] = true
		} else {
			unknownVersionNodes++
		}
	}

	var message strings.Builder

	for major, minors := range r.state.knownVersions {
		checkWithinOneMajor(major, minors, &message)
		checkWithPreviousMajors(major, r.state.knownVersions, &message)
	}

	if message.Len() > 0 {
		return fmt.Errorf(
			`your invocation introduced incompatibility between nodes. Nodes must not differ by more than one major.
			Please STOP restarting and check the connectivity between nodes on different versions.
			Triggered this check: %s.
			Range of versions found: %v.
			If you are absolutely sure in what you are doing, see --suppress-compat-check`,
			message.String(), r.state.knownVersions,
		)
	}

	if unknownVersionNodes > 0 {
		r.logger.Warnf(
			`No incompatible versions have been detected. However, %v nodes reported unparsable versions.
			You may have introduced incompatible versions to the cluster`, unknownVersionNodes,
		)
	}

	return nil
}

func (r *Rolling) logTask(task cms.MaintenanceTask) {
	r.logger.Debugf("Maintenance task result:\n%s", prettyprint.TaskToString(task))
}

func (r *Rolling) logCompleteResult(result *Ydb_Maintenance.ManageActionResult) {
	if result == nil {
		return
	}

	r.logger.Debugf("Manage action result:\n%s", prettyprint.ResultToString(result))
}
