package restart

import (
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

type RestartOptions struct {
	*command.BaseOptions
	*rolling.RollingRestartOptions
}
