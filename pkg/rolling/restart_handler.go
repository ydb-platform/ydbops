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

type queueItem struct {
	value *Ydb_Maintenance.ActionGroupStates
	wg    *sync.WaitGroup
}

type restartHandler struct {
	ctx        context.Context
	logger     *zap.SugaredLogger
	queue      chan queueItem
	batchQueue chan []*Ydb_Maintenance.ActionGroupStates
	restarter  restarters.Restarter
	statusCh   chan<- restartStatus

	// TODO(shmel1k@): probably, not needed here.
	nodes map[uint32]*Ydb_Maintenance.Node

	wg sync.WaitGroup

	nodesInflight        int
	delayBetweenRestarts time.Duration
}

func (rh *restartHandler) push(states []*Ydb_Maintenance.ActionGroupStates) {
	select {
	case <-rh.ctx.Done():
		return
	case rh.batchQueue <- states:
	}
}

func (rh *restartHandler) run() {
	go rh.processQueue()

	go func() {
		for {
			select {
			case <-rh.ctx.Done():
				return
			case batch, ok := <-rh.batchQueue:
				if !ok {
					return
				}
				wg := &sync.WaitGroup{}
				for _, s := range batch {
					wg.Add(1)
					// the purpose of this select statement is to avoid sending more nodes to restart if the context is canceled during a batch processing
					select {
					case <-rh.ctx.Done():
						return
					case rh.queue <- queueItem{value: s, wg: wg}:
					}
				}

				// waits until the whole batch is processed
				wg.Wait()
			}
		}
	}()
}

func (rh *restartHandler) processQueue() {
	for i := 0; i < rh.nodesInflight; i++ {
		rh.wg.Add(1)
		go func() {
			defer rh.wg.Done()

			for {
				select {
				case <-rh.ctx.Done():
					return
				case qItem, ok := <-rh.queue:
					if !ok {
						return
					}

					gs := qItem.value

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

					qItem.wg.Done()

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
	close(rh.batchQueue)
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
		queue:                make(chan queueItem),
		batchQueue:           make(chan []*Ydb_Maintenance.ActionGroupStates),
		statusCh:             statusCh,
		nodesInflight:        nodesInflight,
		nodes:                nodes,
		delayBetweenRestarts: delayBetweenRestarts,
	}
}
