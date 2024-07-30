package list

import (
	"fmt"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

type Options struct{}

func (o *Options) Run(f cmdutil.Factory) error {
	userSID, err := f.GetDiscoveryClient().WhoAmI()
	if err != nil {
		return err
	}

	tasks, err := f.GetCMSClient().MaintenanceTasks(userSID)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Println("There are no maintenance tasks, associated with your user, at the moment.")
	} else {
		for _, task := range tasks {
			taskInfo := prettyprint.TaskToString(task)
			fmt.Println(taskInfo)
		}
	}

	return nil
}
