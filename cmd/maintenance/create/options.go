package create

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"

	"github.com/ydb-platform/ydbops/cmd/restart"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

type Options struct {
	options.FilteringOptions

	MaintenanceDuration int
}

const (
	DefaultMaintenanceDurationSeconds = 3600
)

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	o.FilteringOptions.DefineFlags(fs)

	fs.IntVar(&o.MaintenanceDuration, "duration", DefaultMaintenanceDurationSeconds,
		`CMS will release the node for maintenance for duration seconds. Any maintenance
after that would be considered a regular cluster failure`)
}

func (o *Options) Validate() error {
	if o.MaintenanceDuration < 0 {
		return fmt.Errorf("specified invalid maintenance duration: %d. Must be positive", o.MaintenanceDuration)
	}

	return o.FilteringOptions.Validate()
}

func (o *Options) nodeIdsToNodes(
	nodes []*Ydb_Maintenance.Node,
	nodeIds []uint32,
) []*Ydb_Maintenance.Node {
	targetedNodes := []*Ydb_Maintenance.Node{}

	// TODO @jorres arguments to PrepareRestarters are a dirty hack.
	// We actually only need Filter component from restarters. 2 and 3 arguments
	// are required in PrepareRestarters to actually perform node restarts,
	// but we only use restarters in the scope of this function to filter nodes
	// so their value does not matter. Splitting something like 'Filterers' from
	// Restarters into separate interface should solve this.
	storageRestarter, tenantRestarter := restart.PrepareRestarters(
		&o.FilteringOptions,
		[]string{},
		"",
		o.MaintenanceDuration,
	)

	filterNodeParams := restarters.FilterNodeParams{
		Version:             o.VersionSpec,
		SelectedTenants:     o.TenantList,
		SelectedNodeIds:     nodeIds,
		SelectedHosts:       []string{},
		SelectedDatacenters: o.Datacenters,
		StartedTime:         o.StartedTime,
		ExcludeHosts:        o.ExcludeHosts,
		MaxStaticNodeID:     uint32(o.MaxStaticNodeID),
	}

	clusterNodesInfo := restarters.ClusterNodesInfo{
		AllNodes:        nodes,
		TenantToNodeIds: utils.PopulateTenantToNodesMapping(nodes),
	}

	targetedNodes = append(targetedNodes, storageRestarter.Filter(filterNodeParams, clusterNodesInfo)...)
	targetedNodes = append(targetedNodes, tenantRestarter.Filter(filterNodeParams, clusterNodesInfo)...)

	return targetedNodes
}

func (o *Options) Run(f cmdutil.Factory) error {
	taskUID := cms.TaskUuidPrefix + uuid.New().String()
	duration := time.Duration(o.MaintenanceDuration) * time.Second

	nodes, err := f.GetCMSClient().Nodes()
	if err != nil {
		return err
	}
	nodeIds, errIds := utils.GetNodeIds(o.Hosts)
	hostFQDNs, errFqdns := utils.GetNodeFQDNs(o.Hosts)
	if errIds != nil && errFqdns != nil {
		return fmt.Errorf(
			"failed to parse --hosts argument as node ids (%w) or host fqdns (%w)",
			errIds,
			errFqdns,
		)
	}

	var task cms.MaintenanceTask
	if errIds == nil {
		task, err = f.GetCMSClient().CreateMaintenanceTask(cms.MaintenanceTaskParams{
			Nodes:            o.nodeIdsToNodes(nodes, nodeIds),
			Duration:         durationpb.New(duration),
			AvailabilityMode: o.GetAvailabilityMode(),
			ScopeType:        cms.NodeScope,
			TaskUID:          taskUID,
		})
	} else {
		task, err = f.GetCMSClient().CreateMaintenanceTask(cms.MaintenanceTaskParams{
			Hosts:            hostFQDNs,
			Duration:         durationpb.New(duration),
			AvailabilityMode: o.GetAvailabilityMode(),
			ScopeType:        cms.HostScope,
			TaskUID:          taskUID,
		})
	}

	if err != nil {
		return err
	}

	fmt.Printf(
		"Your task id is:\n\n%s\n\nPlease write it down for refreshing and completing the task later.\n",
		task.GetTaskUid(),
	)

	fmt.Println(prettyprint.TaskToString(task))

	return nil
}
