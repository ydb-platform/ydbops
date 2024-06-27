package rolling

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type NodeType int

const (
	NodeTypeStorage NodeType = 1
	NodeTypeTenant  NodeType = 2
)

type Rolling struct {
	cms       *client.Cms
	discovery *client.Discovery

	logger    *zap.SugaredLogger
	state     *state
	opts      options.RestartOptions
	restarter restarters.Restarter

	nodeType NodeType
}

type state struct {
	nodes               map[uint32]*Ydb_Maintenance.Node
	inactiveNodes       map[uint32]*Ydb_Maintenance.Node
	tenantNameToNodeIds map[string][]uint32
	retriesMadeForNode  map[uint32]int
	tenants             []string
	userSID             string
}

const (
	RestartTaskPrefix = "rolling-restart-"
)

func ExecuteRolling(
	restartOpts options.RestartOptions,
	logger *zap.SugaredLogger,
	restarter restarters.Restarter,
	nodeType NodeType,
) error {
	cmsClient := client.GetCmsClient()
	discoveryClient := client.GetDiscoveryClient()

	r := &Rolling{
		cms:       cmsClient,
		discovery: discoveryClient,
		logger:    logger,
		opts:      restartOpts,
		restarter: restarter,
		nodeType:  nodeType,
	}

	var err error
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

type ProgressMessage struct {
	err error
	restartedNode uint32
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

	activeNodes := []*Ydb_Maintenance.Node{}
	for _, node := range nodesToRestart {
		if _, present := r.state.inactiveNodes[node.NodeId]; present {
			r.logger.Warn(
				"the node with nodeId: %v and host: %s is currently down and will be excluded from restart",
				node.Host,
				node.NodeId,
			)
		} else {
			activeNodes = append(activeNodes, node)
		}
	}

	if len(activeNodes) == 0 {
		r.logger.Warn("There are no nodes that satisfy the specified filters")
		return nil
	}

	tasksPerTenant := make(map[string][]client.MaintenanceTaskParams)
	nodesPerTenant := make(map[string]int)

	if r.nodeType == NodeTypeStorage {
		tasksPerTenant["storage"] = []client.MaintenanceTaskParams{r.makeTaskParams(nodesToRestart)}
	} else {
		for tenantName, tenantNodeIds := range r.state.tenantNameToNodeIds {
			curTenantTasks := []client.MaintenanceTaskParams{}

			cnt := 0
			curTaskNodes := []*Ydb_Maintenance.Node{}
			for _, nodeId := range tenantNodeIds {
				cnt++
				curTaskNodes = append(curTaskNodes, r.state.nodes[nodeId])
				if cnt == 10 { // TODO r.opts.MaxPerTenant
					curTenantTasks = append(curTenantTasks, r.makeTaskParams(curTaskNodes))
					cnt = 0
					curTaskNodes = []*Ydb_Maintenance.Node{}
				}
			}
			if cnt > 0 {
				curTenantTasks = append(curTenantTasks, r.makeTaskParams(curTaskNodes))
			}

			tasksPerTenant[tenantName] = curTenantTasks
			nodesPerTenant[tenantName] = len(tenantNodeIds)
		}
	}

	resultChannel := make(chan ProgressMessage, 1024)

	wg := sync.WaitGroup{}
	for _, sequentialTasks := range tasksPerTenant {
		wg.Add(1)
		go func() {
			r.cmsWaitingLoop(sequentialTasks, resultChannel)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	for {
		newlyRestartedPerBase := make(map[string][]uint32)
		select {
		case msg, ok := <-resultChannel:
			if !ok {
				return nil
			}
			if msg.err != nil {
				return err
			}

			node := r.state.nodes[msg.restartedNode]
			var tenant string
			if node.GetStorage() != nil {
				tenant = "domain"
			} else {
				tenant = node.GetDynamic().GetTenant()
			}

			if _, ok := newlyRestartedPerBase[tenant]; !ok {
				newlyRestartedPerBase[tenant] = []uint32{}
			}
			newlyRestartedPerBase[tenant] = append(newlyRestartedPerBase[tenant], msg.restartedNode)
		default:
			r.logger.Info(prettyprint.AggregateByAllTenants(newlyRestartedPerBase, nodesPerTenant))

			defaultDelay := time.Duration(r.opts.CMSQueryInterval) * time.Second
			time.Sleep(defaultDelay)
		}
	}
}

func (r *Rolling) makeTaskParams(nodes []*Ydb_Maintenance.Node) client.MaintenanceTaskParams {
	return client.MaintenanceTaskParams{
		TaskUID:          RestartTaskPrefix + uuid.New().String(),
		AvailabilityMode: r.opts.GetAvailabilityMode(),
		Duration:         r.opts.GetRestartDuration(),
		Nodes:            nodes,
	}
}

func (r *Rolling) DoRestartPrevious() error {
	return fmt.Errorf("--continue behavior not implemented yet")
}

type TaskState struct {
	m                  *sync.Mutex
	restartedNodes     int
	retriesMadeForNode map[uint32]int

	unreportedButCompletedActionIds []*Ydb_Maintenance.ActionUid
	unreportedButRestartedNodeIds   []uint32
}

func (r *Rolling) cmsWaitingLoop(tasksParams []client.MaintenanceTaskParams, resultChannel chan<- ProgressMessage) {
	for _, taskParams := range tasksParams {
		task, err := r.cms.CreateMaintenanceTask(taskParams)
		if err != nil {
			resultChannel <- ProgressMessage{
				err: err,
			}
			return
		}

		var (
			delay        time.Duration
			taskID       = task.GetTaskUid()
			defaultDelay = time.Duration(r.opts.CMSQueryInterval) * time.Second
		)

		taskState := &TaskState{
			restartedNodes:                  0,
			retriesMadeForNode:              make(map[uint32]int),
			m:                               new(sync.Mutex),
			unreportedButCompletedActionIds: []*Ydb_Maintenance.ActionUid{},
			unreportedButRestartedNodeIds:   []uint32{},
		}

		// r.logger.Infof("Maintenance task %v, processing loop started", taskID)
		for {
			delay = defaultDelay

			// r.logTask(task)

			if task.GetRetryAfter() != nil {
				retryTime := task.GetRetryAfter().AsTime()
				// r.logger.Debugf("Task has retry after attribute: %s", retryTime.Format(time.DateTime))

				if retryDelay := retryTime.Sub(time.Now().UTC()); defaultDelay < retryDelay {
					delay = defaultDelay
				}
			}

			if completed := r.processActionGroupStates(
				task.GetActionGroupStates(),
				taskState,
				resultChannel,
			); completed {
				break
			}

			// r.logger.Infof("Wait next %s delay. Total node progress: %v out of %v", delay, restartedNodes, totalNodes)
			time.Sleep(delay)

			// r.logger.Infof("Refresh maintenance task with id: %s", taskID)
			task, err = r.cms.RefreshMaintenanceTask(taskID)
			// TODO wtf?
			// if err != nil {
			// r.logger.Warnf("Failed to refresh maintenance task: %+v", err)
			// }
		}

		// r.logger.Infof("Maintenance task processing loop completed")
	}
}

func (r *Rolling) processActionGroupStates(
	actions []*Ydb_Maintenance.ActionGroupStates,
	taskState *TaskState,
	resultChan chan<- ProgressMessage,
) bool {
	// r.logger.Debugf("Unfiltered ActionGroupStates: %v", actions)
	performed := collections.FilterBy(actions,
		func(gs *Ydb_Maintenance.ActionGroupStates) bool {
			return gs.ActionStates[0].Status == Ydb_Maintenance.ActionState_ACTION_STATUS_PERFORMED
		},
	)

	if len(performed) == 0 {
		// r.logger.Info("No actions can be taken yet, waiting for CMS to move some actions to PERFORMED...")
		return false
	}

	// r.logger.Infof("%d ActionGroupStates moved to PERFORMED, will restart now...", len(performed))

	wg := new(sync.WaitGroup)

	for _, gs := range performed {
		var (
			as   = gs.ActionStates[0]
			lock = as.Action.GetLockAction()
			node = r.state.nodes[lock.Scope.GetNodeId()]
		)

		// TODO I have a bad feeling about proto comparison
		if collections.Contains(taskState.unreportedButCompletedActionIds, as.ActionUid) {
			continue
			// r.logger.Debugf(
			// 	"Node id %v already restarted, but CompleteAction failed on last iteration, "+
			// 		"so CMS does not know it is complete yet.",
			// 	node.NodeId,
			// )
		}

		r.logger.Debugf("Drain node with id: %d", node.NodeId)
		wg.Add(1)

		go func() {
			defer wg.Done()

			// TODO: drain node, but public draining api is not available yet
			r.logger.Warn("DRAINING NOT IMPLEMENTED YET")

			r.logger.Debugf("Restart node with id: %d", node.NodeId)
			if err := r.restarter.RestartNode(node); err != nil {
				retriesUntilNow := taskState.getAndIncrement(node.NodeId)

				// r.logger.Warnf(
				// 	"Failed to restart node with id: %d, attempt number %v, because of: %s",
				// 	node.NodeId,
				// 	retriesUntilNow,
				// 	err.Error(),
				// )

				if retriesUntilNow+1 == r.opts.RestartRetryNumber {
					taskState.markActionComplete(as.ActionUid, node.NodeId)
					// r.logger.Warnf("Failed to retry node %v specified number of times (%v)", node.NodeId, r.opts.RestartRetryNumber)
				}
			} else {
				taskState.markActionComplete(as.ActionUid, node.NodeId)
				// r.logger.Debugf("Successfully restarted node with id: %d", node.NodeId)
			}
		}()
	}

	wg.Wait()

	result, err := r.cms.CompleteAction(taskState.unreportedButCompletedActionIds)
	if err != nil {
		r.logger.Warnf("Failed to complete action: %+v", err)
		return false
	}

	taskState.reportCompletedActions(resultChan)

	return len(actions) == len(result.ActionStatuses)
}

func (t *TaskState) markActionComplete(actionUid *Ydb_Maintenance.ActionUid, nodeId uint32) {
	t.m.Lock()
	defer t.m.Unlock()

	t.unreportedButCompletedActionIds = append(t.unreportedButCompletedActionIds, actionUid)
	t.unreportedButRestartedNodeIds = append(t.unreportedButRestartedNodeIds, nodeId)
}

func (t *TaskState) reportCompletedActions(resultChan chan<- ProgressMessage) {
	for _, nodeId := range t.unreportedButRestartedNodeIds {
		resultChan <- ProgressMessage{
			restartedNode: nodeId,
		}
	}

	t.unreportedButCompletedActionIds = []*Ydb_Maintenance.ActionUid{}
	t.unreportedButRestartedNodeIds = []uint32{}
}

func (t *TaskState) getAndIncrement(nodeId uint32) int {
	t.m.Lock()
	defer t.m.Unlock()

	retries := t.retriesMadeForNode[nodeId]
	t.retriesMadeForNode[nodeId]++
	return retries
}

func (r *Rolling) populateTenantToNodesMapping(nodes []*Ydb_Maintenance.Node) map[string][]uint32 {
	tenantNameToNodeIds := make(map[string][]uint32)
	for _, node := range nodes {
		dynamicNode := node.GetDynamic()
		if dynamicNode != nil {
			tenantNameToNodeIds[dynamicNode.GetTenant()] = append(
				tenantNameToNodeIds[dynamicNode.GetTenant()],
				node.NodeId,
			)
		}
	}

	return tenantNameToNodeIds
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
		tenantNameToNodeIds: r.populateTenantToNodesMapping(activeNodes),
		tenants:             tenants,
		userSID:             userSID,
		nodes:               collections.ToMap(activeNodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
		inactiveNodes:       collections.ToMap(inactiveNodes, func(n *Ydb_Maintenance.Node) uint32 { return n.NodeId }),
		retriesMadeForNode:  make(map[uint32]int),
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

func (r *Rolling) logTask(task client.MaintenanceTask) {
	r.logger.Debugf("Maintenance task result:\n%s", prettyprint.TaskToString(task))
}

func logCompleteResult(logger *zap.SugaredLogger, result *Ydb_Maintenance.ManageActionResult) {
	if result == nil {
		return
	}

	logger.Debugf("Manage action result:\n%s", prettyprint.ResultToString(result))
}

- check prod grafana
- continue multi tenant coding testing
- maintenance api recording
