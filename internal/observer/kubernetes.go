package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesWatcher struct {
	clientset *kubernetes.Clientset
	namespace string
	db        *storage.PostgresClient
	enabled   bool
	logger    *zap.Logger
}

func NewKubernetesWatcher(namespace string, db *storage.PostgresClient, logger *zap.Logger) (*KubernetesWatcher, error) {
	if namespace == "" {
		namespace = "default"
	}

	watcher := &KubernetesWatcher{
		namespace: namespace,
		db:        db,
		enabled:   false,
		logger:    logger,
	}

	clientset, err := watcher.createKubernetesClient()
	if err != nil {
		logger.Warn("Could not connect to Kubernetes, watcher disabled", zap.Error(err))
		return watcher, nil
	}

	watcher.clientset = clientset
	watcher.enabled = true 

	return watcher, nil
}

func (k *KubernetesWatcher) createKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig() 
	/* 
	"Hey Kubernetes… am I already running inside your cluster as a pod?"
	If answer = YES (we are inside K8s):
	then Kubernetes will automatically give you the credentials (token, certs)
	and this function gives you a ready config to talk to API server.
	→ then we return clientset
	*/
	if err == nil {
		return kubernetes.NewForConfig(config) // We are Making the clinetset Object that csan perform tasks that the kubernetes need to perfoem or we want the kubernetes to perform 
	}

	kubeconfigPath := os.Getenv("KUBECONFIG") //Docker Compose mai set kar rakhi hai maine 
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s", kubeconfigPath)
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	return kubernetes.NewForConfig(config) //Yeh Wali Apni config hai jo hamne register ki hai khud se 
}

func (k *KubernetesWatcher) Start(ctx context.Context) error {
	if !k.enabled {
		<-ctx.Done()
		return nil
	}

	go k.watchPods(ctx)
	go k.collectPodMetrics(ctx)

	<-ctx.Done()
	return ctx.Err()
	// wait until context is cancelled when cancelled → return the reason why it cancelled
}

func (k *KubernetesWatcher) watchPods(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := k.watchPodsOnce(ctx); err != nil {
				k.logger.Error("Pod watch error", zap.Error(err))
				time.Sleep(10 * time.Second)
			}
		}
	}
}

func (k *KubernetesWatcher) watchPodsOnce(ctx context.Context) error {
	timeout := int64(300)
	watcher, err := k.clientset.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to watch pods: %w", err)
	} // 
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan(): // This is Giving Kubernetes Event 
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			if err := k.handlePodEvent(ctx, event); err != nil {
				k.logger.Error("Pod event error", zap.Error(err))
			}
		}
	}
}

func (k *KubernetesWatcher) handlePodEvent(ctx context.Context, event watch.Event) error {
	pod, ok := event.Object.(*corev1.Pod)
	/*
	Because Kubernetes watch can send different types of objects.
	So before using, we must check:	
	Example things that cause events:
	a pod got created.
	a pod got deleted.
	a pod crashed.
	image pull failed.
	container restarted.
	scheduler moved a pod. 
	*/ 
	if !ok {
		return fmt.Errorf("unexpected object type: %T", event.Object)
	}

	eventType := string(event.Type)
	message := k.buildEventMessage(pod, eventType)

	storageEvent := &storage.Event{
		Timestamp: time.Now(),
		EventType: eventType,
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		Message:   message,
	} //Yeh Sabh de diya maine 

	if err := k.db.SaveEvent(ctx, storageEvent); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	restarts := k.getPodRestarts(pod)
	if restarts >= 3 {
		k.logger.Warn("Pod crash-looping",
			zap.String("pod", pod.Name),
			zap.Int32("restarts", restarts),
		)

		crashEvent := &storage.Event{
			Timestamp: time.Now(),
			EventType: "CrashLoop",
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Message:   fmt.Sprintf("Pod restarted %d times", restarts),
		}
		_ = k.db.SaveEvent(ctx, crashEvent)
	}

	return nil
}

func (k *KubernetesWatcher) collectPodMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	if err := k.collectAndStorePodMetrics(ctx); err != nil {
		k.logger.Error("Initial pod metrics collection failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := k.collectAndStorePodMetrics(ctx); err != nil {
				k.logger.Error("Pod metrics error", zap.Error(err))
			}
		}
	}
}

func (k *KubernetesWatcher) collectAndStorePodMetrics(ctx context.Context) error {
	pods, err := k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var metrics []*storage.Metric

	for _, pod := range pods.Items {
		// Pod status metric
		statusMetric := &storage.Metric{
			Timestamp:   time.Now(),
			ServiceName: pod.Name,
			MetricName:  "pod_status",
			MetricValue: k.getPodStatusValue(&pod),
			Labels:      k.buildPodLabels(&pod),
		}
		metrics = append(metrics, statusMetric)

		// Restart count metric
		restartMetric := &storage.Metric{
			Timestamp:   time.Now(),
			ServiceName: pod.Name,
			MetricName:  "pod_restarts",
			MetricValue: float64(k.getPodRestarts(&pod)),
			Labels:      k.buildPodLabels(&pod),
		}
		metrics = append(metrics, restartMetric)
	}

	if len(metrics) > 0 {
		if err := k.db.BatchSaveMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("failed to save pod metrics: %w", err)
		}
	}

	return nil
}

func (k *KubernetesWatcher) buildEventMessage(pod *corev1.Pod, eventType string) string {
	switch eventType {
	case "ADDED":
		return fmt.Sprintf("Pod %s created (phase: %s)", pod.Name, pod.Status.Phase)
	case "MODIFIED":
		return fmt.Sprintf("Pod %s updated (phase: %s, restarts: %d)",
			pod.Name, pod.Status.Phase, k.getPodRestarts(pod))
	case "DELETED":
		return fmt.Sprintf("Pod %s deleted", pod.Name)
	default:
		return fmt.Sprintf("Pod %s event: %s", pod.Name, eventType)
	}
} // Build Event Messages 

func (k *KubernetesWatcher) getPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

func (k *KubernetesWatcher) getPodStatusValue(pod *corev1.Pod) float64 {
	switch pod.Status.Phase {
	case corev1.PodPending:
		return 0.0
	case corev1.PodRunning:
		if k.isPodReady(pod) {
			return 1.0 // Running and ready
		}
		return 0.5 // Running but not ready
	case corev1.PodSucceeded:
		return 2.0
	case corev1.PodFailed:
		return -1.0
	case corev1.PodUnknown:
		return -2.0
	default:
		return -3.0
	}
}

// isPodReady checks if pod is ready
func (k *KubernetesWatcher) isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (k *KubernetesWatcher) buildPodLabels(pod *corev1.Pod) json.RawMessage {
	labels := map[string]interface{}{
		"namespace": pod.Namespace,
		"phase":     string(pod.Status.Phase),
		"ready":     k.isPodReady(pod),
		"restarts":  k.getPodRestarts(pod),
		"node":      pod.Spec.NodeName,
	}

	data, _ := json.Marshal(labels)
	return data
}

func (k *KubernetesWatcher) Health(ctx context.Context) error {
	if !k.enabled {
		return nil
	}

	_, err := k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{
		Limit: 1,
	})
	if err != nil {
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}

	return nil
}

func (k *KubernetesWatcher) GetPodMetrics(ctx context.Context) ([]PodMetric, error) {
	if !k.enabled {
		return nil, fmt.Errorf("kubernetes watcher not enabled")
	}

	pods, err := k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	metrics := make([]PodMetric, 0, len(pods.Items))
	for _, pod := range pods.Items {
		metrics = append(metrics, PodMetric{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Ready:     k.isPodReady(&pod),
			Restarts:  k.getPodRestarts(&pod),
		})
	}

	return metrics, nil
}

type PodMetric struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Phase     string `json:"phase"`
	Ready     bool   `json:"ready"`
	Restarts  int32  `json:"restarts"`
}
