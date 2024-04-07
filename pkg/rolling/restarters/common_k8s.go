package restarters

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ydb-platform/ydbops/pkg/options"
)

const (
	DefaultPodPhasePollingInterval = time.Second * 10
)

type k8sRestarter struct {
	k8sClient     *kubernetes.Clientset
	FQDNToPodName map[string]string
	logger        *zap.SugaredLogger
}

func newK8sRestarter(logger *zap.SugaredLogger) k8sRestarter {
	return k8sRestarter{
		k8sClient:     nil, // initialized later
		FQDNToPodName: make(map[string]string),
		logger:        logger,
	}
}

func (r *k8sRestarter) createK8sClient(kubeconfigPath string) *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		r.logger.Fatalf("Failed to build config from flags %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func (r *k8sRestarter) waitPodRunning(
	namespace, podName string,
	oldUID types.UID,
	podRestartTimeout time.Duration,
) error {
	checkInterval := DefaultPodPhasePollingInterval
	start := time.Now()
	for {
		if time.Since(start) >= podRestartTimeout {
			return fmt.Errorf("timed out waiting for a pod %s to start", podName)
		}

		pod, err := r.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})

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

func (r *k8sRestarter) prepareK8sState(kubeconfigPath, labelSelector, namespace string) {
	r.k8sClient = r.createK8sClient(kubeconfigPath)

	pods, err := r.k8sClient.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)

	for i := 0; i < len(pods.Items); i++ {
		pod := &pods.Items[i]
		fullPodFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", pod.Spec.Hostname, pod.Spec.Subdomain, pod.Namespace)
		r.FQDNToPodName[fullPodFQDN] = pod.Name
	}

	if err != nil {
		panic(err.Error()) // TODO refactor Filter. Filter should also return error, it makes sense
	}
}

func (r *k8sRestarter) restartNodeByRestartingPod(nodeFQDN, namespace string) error {
	podName := nodeFQDN
	if _, present := r.FQDNToPodName[nodeFQDN]; present {
		podName = r.FQDNToPodName[nodeFQDN]
	}

	r.logger.Infof("Restarting node %s on the %s pod", nodeFQDN, podName)

	pod, err := r.k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("pod scheduled for deletion %s not found: %w", podName, err)
	}

	oldUID := pod.ObjectMeta.UID

	err = r.k8sClient.CoreV1().Pods(namespace).Delete(
		context.TODO(),
		podName,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}

	return r.waitPodRunning(
		namespace,
		podName,
		oldUID,
		time.Duration(options.RestartOptionsInstance.RestartDuration)*time.Second,
	)
}
