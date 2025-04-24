package restarters

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ContainerStorageName = "ydb-storage"
	ContainerDynnodeName = "ydb-dynamic"
	PortInterconnectName = "interconnect"
)

type k8sRestarter struct {
	k8sClient     *kubernetes.Clientset
	FQDNToPodName map[string]string
	logger        *zap.SugaredLogger
	options       *k8sRestarterOptions
}

type k8sRestarterOptions struct {
	restartDuration time.Duration
}

func newK8sRestarter(logger *zap.SugaredLogger, params *k8sRestarterOptions) k8sRestarter {
	return k8sRestarter{
		k8sClient:     nil, // initialized later
		FQDNToPodName: make(map[string]string),
		logger:        logger,
		options:       params,
	}
}

func (r *k8sRestarter) createK8sClient(kubeconfigPath string) *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		r.logger.Fatalf(
			"Failed to build kubeconfig from kubeconfig file %s: %s",
			kubeconfigPath,
			err.Error(),
		)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		r.logger.Fatalf("Failed to create a k8s client from config: %s", err.Error())
	}

	return clientset
}

func podInterconnectPort(p *v1.Pod) int32 {
	for _, container := range p.Spec.Containers {
		if container.Name != ContainerStorageName && container.Name != ContainerDynnodeName {
			continue
		}
		for _, port := range container.Ports {
			if port.Name == PortInterconnectName {
				return port.ContainerPort
			}
		}
	}
	return -1
}

func (r *k8sRestarter) prepareK8sState(kubeconfigPath, labelSelector, namespace string) {
	r.k8sClient = r.createK8sClient(kubeconfigPath)

	pods, err := r.k8sClient.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)

	for _, pod := range pods.Items {
		fullPodFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", pod.Spec.Hostname, pod.Spec.Subdomain, pod.Namespace)
		r.FQDNToPodName[pod.Name] = pod.Name
		r.FQDNToPodName[fullPodFQDN] = pod.Name
		r.FQDNToPodName[pod.Spec.NodeName] = pod.Name
		if icPort := podInterconnectPort(&pod); icPort != -1 {
			r.FQDNToPodName[fmt.Sprintf("%s:%d", pod.Name, icPort)] = pod.Name
			r.FQDNToPodName[fmt.Sprintf("%s:%d", fullPodFQDN, icPort)] = pod.Name
			r.FQDNToPodName[fmt.Sprintf("%s:%d", pod.Spec.NodeName, icPort)] = pod.Name
		}
	}

	if err != nil {
		panic(err.Error()) // TODO refactor Filter. Filter should also return error, it makes sense
	}
}

func (r *k8sRestarter) restartNodeByRestartingPod(nodeFQDN string, icPort uint32, namespace string) error {
	podName, present := r.FQDNToPodName[fmt.Sprintf("%s:%d", nodeFQDN, icPort)]
	if !present {
		podName, present = r.FQDNToPodName[nodeFQDN]
	}
	if !present {
		return fmt.Errorf(
			"failed to determine which pod corresponds to node fqdn %s port %d\n"+
				"This is most likely a bug, contact the developers.\n"+
				"If possible, attach logs from invocation with --verbose flag", nodeFQDN, icPort)
	}

	r.logger.Infof("Restarting pod %s on the %s node", podName, nodeFQDN)

	pod, err := r.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("pod scheduled for deletion %s not found: %w", podName, err)
	}

	oldUID := pod.UID
	r.logger.Debugf("Pod %s id: %v", podName, oldUID)

	err = r.k8sClient.CoreV1().Pods(namespace).Delete(
		context.TODO(),
		podName,
		metav1.DeleteOptions{},
	)

	return err
}
