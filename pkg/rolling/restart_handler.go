package rolling

import (
	"context"
	"sync"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type restartStatus struct {
	nodeID uint32
	as     *Ydb_Maintenance.ActionState
	err    error
}

type restartHandler struct {
	ctx       context.Context
	logger    *zap.SugaredLogger
	queue     chan *Ydb_Maintenance.ActionGroupStates
	restarter restarters.Restarter
	statusCh  chan<- restartStatus

	// TODO(shmel1k@): probably, not needed here.
	nodes map[uint32]*Ydb_Maintenance.Node

	wg sync.WaitGroup

	nodesInflight        int
	delayBetweenRestarts time.Duration
}

func (rh *restartHandler) push(state *Ydb_Maintenance.ActionGroupStates) {
	select {
	case <-rh.ctx.Done():
	case rh.queue <- state:
	}
}

func (rh *restartHandler) run() {
	for i := 0; i < rh.nodesInflight; i++ {
		rh.wg.Add(1)
		go func() {
			defer rh.wg.Done()

			for {
				select {
				case <-rh.ctx.Done():
					return
				case gs, ok := <-rh.queue:
					if !ok {
						return
					}

					var (
						as   = gs.ActionStates[0]
						lock = as.Action.GetLockAction()
						node = rh.nodes[lock.Scope.GetNodeId()]
					)

					rh.logger.Debugf("Restart node with id: %d", node.GetNodeId())

					// TODO(shmel1k@): draining should be implemented in RestartNode.
					rh.logger.Debugf("Drain node with id: %d", node.GetNodeId())
					// TODO: drain node, but public draining api is not available yet
					rh.logger.Info("DRAINING NOT IMPLEMENTED YET")

					err := rh.restarter.RestartNode(node)
					rh.statusCh <- restartStatus{
						nodeID: lock.Scope.GetNodeId(),
						as:     as,
						err:    err,
					}

					select {
					case <-rh.ctx.Done():
						return
					case <-time.After(rh.delayBetweenRestarts):
						continue
					}
				}
			}
		}()
	}
}

func (rh *restartHandler) stop(waitForDelay bool) {
	close(rh.queue)
	if waitForDelay {
		rh.wg.Wait()
	}
}

func newRestartHandler(
	ctx context.Context,
	logger *zap.SugaredLogger,
	restarter restarters.Restarter,
	nodesInflight int,
	delayBetweenRestarts time.Duration,
	nodes map[uint32]*Ydb_Maintenance.Node,
	statusCh chan<- restartStatus,
) *restartHandler {
	return &restartHandler{
		ctx:                  ctx,
		logger:               logger,
		restarter:            restarter,
		queue:                make(chan *Ydb_Maintenance.ActionGroupStates),
		statusCh:             statusCh,
		nodesInflight:        nodesInflight,
		nodes:                nodes,
		delayBetweenRestarts: delayBetweenRestarts,
	}
}
