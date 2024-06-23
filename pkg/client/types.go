package client

import "github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"

type OperationResponse interface {
	GetOperation() *Ydb_Operations.Operation
}
