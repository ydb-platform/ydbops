package client

import (
	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"google.golang.org/protobuf/types/known/durationpb"
)

type MaintenanceScopeType int

const (
	NodeScope MaintenanceScopeType = 1
	HostScope MaintenanceScopeType = 2
)

type MaintenanceTaskParams struct {
	TaskUID          string
	AvailabilityMode Ydb_Maintenance.AvailabilityMode
	Priority         int32
	Duration         *durationpb.Duration

	ScopeType MaintenanceScopeType

	Nodes []*Ydb_Maintenance.Node
	Hosts []string
}

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}
