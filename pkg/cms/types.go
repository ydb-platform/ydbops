package cms

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MaintenanceTaskParams struct {
	TaskUid          string
	AvailAbilityMode Ydb_Maintenance.AvailabilityMode
	Duration         *durationpb.Duration
	Nodes            []*Ydb_Maintenance.Node
}

type MaintenanceTask interface {
	GetRetryAfter() *timestamppb.Timestamp
	GetActionGroupStates() []*Ydb_Maintenance.ActionGroupStates
	GetTaskUid() string
}

type maintenanceTaskResult struct {
	TaskUid           string
	ActionGroupStates []*Ydb_Maintenance.ActionGroupStates
}

func (g *maintenanceTaskResult) GetRetryAfter() *timestamppb.Timestamp { return nil }
func (g *maintenanceTaskResult) GetActionGroupStates() []*Ydb_Maintenance.ActionGroupStates {
	return g.ActionGroupStates
}
func (g *maintenanceTaskResult) GetTaskUid() string { return g.TaskUid }
