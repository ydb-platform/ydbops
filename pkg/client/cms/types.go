package cms

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MaintenanceScopeType int

const (
	NodeScope MaintenanceScopeType = 1
	HostScope MaintenanceScopeType = 2
)

type MaintenanceTaskParams struct {
	TaskUID          string
	AvailabilityMode Ydb_Maintenance.AvailabilityMode
	Duration         *durationpb.Duration

	ScopeType MaintenanceScopeType

	Nodes []*Ydb_Maintenance.Node
	Hosts []string
}

type MaintenanceTask interface {
	GetRetryAfter() *timestamppb.Timestamp
	GetActionGroupStates() []*Ydb_Maintenance.ActionGroupStates
	GetTaskUid() string
}

type maintenanceTaskResult struct {
	TaskUID           string
	ActionGroupStates []*Ydb_Maintenance.ActionGroupStates
}

func (g *maintenanceTaskResult) GetRetryAfter() *timestamppb.Timestamp { return nil }
func (g *maintenanceTaskResult) GetActionGroupStates() []*Ydb_Maintenance.ActionGroupStates {
	return g.ActionGroupStates
}
func (g *maintenanceTaskResult) GetTaskUid() string { return g.TaskUID }
