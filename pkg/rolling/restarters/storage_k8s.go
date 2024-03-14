package restarters

import (
	"context"
	"fmt"
	"time"

	"github.com/ydb-platform/ydb-go-genproto/draft/protos/Ydb_Maintenance"
	"github.com/ydb-platform/ydbops/internal/collections"
	"github.com/ydb-platform/ydbops/pkg/options"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultPodPhasePollingInterval = time.Second * 10
)

type StorageK8sRestarter struct {
	Opts      *StorageK8sOpts
	k8sClient *kubernetes.Clientset

	hostnameToPod  map[string]string
	hostnameToNode map[string]string

	logger *zap.SugaredLogger
}

func NewStorageK8sRestarter(logger *zap.SugaredLogger) *StorageK8sRestarter {
	return &StorageK8sRestarter{
		Opts:           &StorageK8sOpts{},
		hostnameToPod:  make(map[string]string),
		hostnameToNode: make(map[string]string),
		logger:         logger,
	}
}

func (r *StorageK8sRestarter) prepareK8sState() {
	config, err := clientcmd.BuildConfigFromFlags("", r.Opts.KubeconfigPath)
	if err != nil {
		r.logger.Fatalf("Failed to build config from flags %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	r.k8sClient = clientset

	labelSelector := "app.kubernetes.io/instance=storage"
	pods, err := r.k8sClient.CoreV1().Pods(r.Opts.Namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)

	for _, pod := range pods.Items {
		r.hostnameToPod[pod.Spec.Hostname] = pod.Name
		r.hostnameToNode[pod.Spec.Hostname] = pod.Spec.NodeName
	}

	r.logger.Debugf("hostnameToPod: %+v", r.hostnameToPod)
	r.logger.Debugf("hostnameToNode: %+v", r.hostnameToNode)

	if err != nil {
		panic(err.Error())
	}
}

func (r StorageK8sRestarter) waitPodRunning(
	podName string,
	oldUID types.UID,
	podRestartTimeout time.Duration,
) error {
	checkInterval := DefaultPodPhasePollingInterval
	start := time.Now()
	for {
		if time.Since(start) >= podRestartTimeout {
			return fmt.Errorf("Timed out waiting for a pod %s to start", podName)
		}

		pod, err := r.k8sClient.CoreV1().Pods(r.Opts.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})

		if pod.ObjectMeta.UID == oldUID {
			r.logger.Debugf("Old pod %s is still not deleted, old UID found", podName)
			time.Sleep(checkInterval)
			continue
		}

		if errors.IsNotFound(err) {
			remainingTime := podRestartTimeout - time.Since(start)
			r.logger.Debugf(
				"Pod %s is not found, will wait %v more seconds",
				podName,
				remainingTime.Seconds(),
			)
			time.Sleep(checkInterval)
			continue
		}

		if pod.Status.Phase == v1.PodRunning {
			r.logger.Debugf("Found pod %s to be restarted and running", podName)
			return nil
		}
	}
}

func (r StorageK8sRestarter) RestartNode(node *Ydb_Maintenance.Node) error {
	podName := r.hostnameToPod[node.Host]
	r.logger.Infof("Restarting node %s on the %s pod", node.Host, podName)

	pod, err := r.k8sClient.CoreV1().Pods(r.Opts.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Pod scheduled for deletion %s not found: %w", podName, err)
	}

	oldUID := pod.ObjectMeta.UID

	err = r.k8sClient.CoreV1().Pods(r.Opts.Namespace).Delete(
		context.TODO(),
		podName,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}

	return r.waitPodRunning(podName, oldUID, time.Duration(
		options.RestartOptionsInstance.RestartDuration,
	))
}

func (r *StorageK8sRestarter) Filter(
	spec FilterNodeParams,
	cluster ClusterNodesInfo,
) []*Ydb_Maintenance.Node {
	r.prepareK8sState()

	allStorageNodes := FilterStorageNodes(cluster.AllNodes)

	selectedNodes := []*Ydb_Maintenance.Node{}
	if len(spec.SelectedNodeIds) > 0 || len(spec.SelectedHostFQDNs) > 0 {
		selectedNodes = append(
			selectedNodes,
			FilterByNodeIds(allStorageNodes, spec.SelectedNodeIds)...,
		)

		for _, node := range allStorageNodes {
			selectedHostFQDNsMap := collections.ToIndexMap(spec.SelectedHostFQDNs)

			if _, present := selectedHostFQDNsMap[node.Host]; present {
				selectedNodes = append(selectedNodes, node)
				continue
			}

			if _, present := selectedHostFQDNsMap[r.hostnameToPod[node.Host]]; present {
				selectedNodes = append(selectedNodes, node)
				continue
			}
		}
	} else {
		selectedNodes = allStorageNodes
	}

	r.logger.Debugf("Storage K8s restarter selected following nodes for restart: %v", selectedNodes)
	return selectedNodes
}
