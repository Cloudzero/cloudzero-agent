package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/parallel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Standard test labels applied to all resources
var standardLabels = map[string]string{
	"test":    "always-allow",
	"team":    "cirrus",
	"purpose": "testing",
}

type Config struct {
	NamespaceCount   int
	PodsPerNamespace int
	LiveDuration     time.Duration
	NSGroup          string
	MaxConcurrency   int
	NamespacePrefix  string
	PodPrefix        string
	BatchSize        int
	CPURequest       string
	CPULimit         string
	MemoryRequest    string
	MemoryLimit      string
	ResourceProfile  string
}

type LoadTester struct {
	client kubernetes.Interface
	config Config
}

func main() {
	config := parseFlags()

	client, err := createKubernetesClient(*flag.String("kubeconfig", "", "Path to kubeconfig file"))
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	tester := &LoadTester{
		client: client,
		config: config,
	}

	ctx := context.Background()
	printTestSummary(config)

	if err := tester.RunLoadTest(ctx); err != nil {
		log.Fatalf("Load test failed: %v", err)
	}
}

func parseFlags() Config {
	var (
		namespaceCount   = flag.Int("namespaces", 100, "Number of namespaces to create")
		podsPerNamespace = flag.Int("pods-per-ns", 1, "Number of pods per namespace")
		liveDuration     = flag.Duration("live-duration", 30*time.Second, "How long resources should live")
		nsGroup          = flag.String("nsgroup", fmt.Sprintf("webhook-load-test-%d", time.Now().Unix()), "NSGroup label value")
		maxConcurrency   = flag.Int("concurrency", 10, "Maximum concurrent operations")
		namespacePrefix  = flag.String("ns-prefix", "webhook-test-ns", "Prefix for namespace names")
		podPrefix        = flag.String("pod-prefix", "webhook-test-pod", "Prefix for pod names")
		batchSize        = flag.Int("batch-size", 0, "Process in waves of this size (0 = all at once)")
		resourceProfile  = flag.String("resource-profile", "light", "Resource profile: light, medium, heavy, or custom")
		cpuRequest       = flag.String("cpu-request", "", "CPU request (only used with custom profile)")
		cpuLimit         = flag.String("cpu-limit", "", "CPU limit (only used with custom profile)")
		memoryRequest    = flag.String("memory-request", "", "Memory request (only used with custom profile)")
		memoryLimit      = flag.String("memory-limit", "", "Memory limit (only used with custom profile)")
	)
	flag.Parse()

	config := Config{
		NamespaceCount:   *namespaceCount,
		PodsPerNamespace: *podsPerNamespace,
		LiveDuration:     *liveDuration,
		NSGroup:          *nsGroup,
		MaxConcurrency:   *maxConcurrency,
		NamespacePrefix:  *namespacePrefix,
		PodPrefix:        *podPrefix,
		BatchSize:        *batchSize,
		ResourceProfile:  *resourceProfile,
		CPURequest:       *cpuRequest,
		CPULimit:         *cpuLimit,
		MemoryRequest:    *memoryRequest,
		MemoryLimit:      *memoryLimit,
	}

	// Apply resource profile
	applyResourceProfile(&config)

	return config
}

func applyResourceProfile(config *Config) {
	switch config.ResourceProfile {
	case "light":
		config.CPURequest = "10m"
		config.CPULimit = "20m"
		config.MemoryRequest = "16Mi"
		config.MemoryLimit = "32Mi"
	case "medium":
		config.CPURequest = "100m"
		config.CPULimit = "200m"
		config.MemoryRequest = "128Mi"
		config.MemoryLimit = "256Mi"
	case "heavy":
		config.CPURequest = "500m"
		config.CPULimit = "1000m"
		config.MemoryRequest = "512Mi"
		config.MemoryLimit = "1Gi"
	case "custom":
		// Use provided values or defaults
		if config.CPURequest == "" {
			config.CPURequest = "50m"
		}
		if config.CPULimit == "" {
			config.CPULimit = "100m"
		}
		if config.MemoryRequest == "" {
			config.MemoryRequest = "32Mi"
		}
		if config.MemoryLimit == "" {
			config.MemoryLimit = "64Mi"
		}
	default:
		fmt.Printf("Warning: Unknown resource profile '%s', using 'light'\n", config.ResourceProfile)
		config.ResourceProfile = "light"
		applyResourceProfile(config) // Recursive call with 'light'
	}
}

func createKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	if kubeconfig != "" {
		clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{},
		)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create client config: %w", err)
	}

	return kubernetes.NewForConfig(restConfig)
}

func printTestSummary(config Config) {
	fmt.Printf("Starting webhook load test:\n")
	fmt.Printf("- Namespaces: %d (prefix: %s)\n", config.NamespaceCount, config.NamespacePrefix)
	fmt.Printf("- Pods per namespace: %d (prefix: %s)\n", config.PodsPerNamespace, config.PodPrefix)
	fmt.Printf("- Live duration: %v\n", config.LiveDuration)
	fmt.Printf("- NSGroup: %s\n", config.NSGroup)
	fmt.Printf("- Max concurrency: %d\n", config.MaxConcurrency)
	if config.BatchSize > 0 {
		fmt.Printf("- Batch size: %d (lifecycle batching)\n", config.BatchSize)
	} else {
		fmt.Printf("- Batch size: all at once\n")
	}
	fmt.Printf("- Resource profile: %s (CPU: %s/%s, Memory: %s/%s)\n",
		config.ResourceProfile, config.CPURequest, config.CPULimit,
		config.MemoryRequest, config.MemoryLimit)
	fmt.Printf("- Total resources: %d\n", config.NamespaceCount+config.NamespaceCount*config.PodsPerNamespace)
	fmt.Println()

	fmt.Printf("Expected namespace names: %s-1 to %s-%d\n", config.NamespacePrefix, config.NamespacePrefix, config.NamespaceCount)
	fmt.Printf("Expected pod names: %s-1 to %s-%d (in each namespace)\n", config.PodPrefix, config.PodPrefix, config.PodsPerNamespace)
	fmt.Println()
}

func (lt *LoadTester) RunLoadTest(ctx context.Context) error {
	if lt.config.BatchSize <= 0 {
		// Original behavior: all resources at once
		return lt.runTraditionalTest(ctx)
	}

	// Lifecycle batching: create → wait → delete → repeat
	return lt.runLifecycleBatchTest(ctx)
}

func (lt *LoadTester) runTraditionalTest(ctx context.Context) error {
	phases := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Creating namespaces", lt.createNamespacesAll},
		{"Creating pods", lt.createPodsWithDelay},
		{"Validating resources", lt.validateResources},
		{"Cleaning up", lt.cleanupAll},
	}

	for i, phase := range phases {
		if i == 2 { // After creating pods, wait before validation
			fmt.Printf("Phase %d: Waiting %v for resources to live...\n", i+1, lt.config.LiveDuration)
			time.Sleep(lt.config.LiveDuration)
		}

		fmt.Printf("Phase %d: %s...\n", i+1, phase.name)
		start := time.Now()

		if err := phase.fn(ctx); err != nil {
			return fmt.Errorf("failed to %s: %w", phase.name, err)
		}

		fmt.Printf("✅ %s completed in %v\n\n", phase.name, time.Since(start))
	}

	fmt.Println("🎉 Load test completed successfully!")
	return nil
}

func (lt *LoadTester) runLifecycleBatchTest(ctx context.Context) error {
	totalBatches := (lt.config.NamespaceCount + lt.config.BatchSize - 1) / lt.config.BatchSize

	fmt.Printf("Running lifecycle batch test: %d batches of %d namespaces each\n", totalBatches, lt.config.BatchSize)
	fmt.Printf("Each batch: create namespaces → create pods → wait %v → cleanup → next batch\n\n", lt.config.LiveDuration)

	overallStart := time.Now()

	for batch := 0; batch < totalBatches; batch++ {
		start := batch * lt.config.BatchSize
		end := start + lt.config.BatchSize
		if end > lt.config.NamespaceCount {
			end = lt.config.NamespaceCount
		}

		batchSize := end - start
		fmt.Printf("🔄 Batch %d/%d: Processing namespaces %d-%d (%d namespaces)\n",
			batch+1, totalBatches, start+1, end, batchSize)

		if err := lt.runSingleBatch(ctx, start+1, end); err != nil {
			return fmt.Errorf("batch %d/%d failed: %w", batch+1, totalBatches, err)
		}

		fmt.Printf("✅ Batch %d/%d completed\n\n", batch+1, totalBatches)
	}

	fmt.Printf("🎉 All %d batches completed successfully in %v!\n", totalBatches, time.Since(overallStart))
	return nil
}

func (lt *LoadTester) runSingleBatch(ctx context.Context, startNS, endNS int) error {
	batchSize := endNS - startNS + 1

	// Phase 1: Create namespaces for this batch
	fmt.Printf("  Phase 1: Creating %d namespaces...\n", batchSize)
	start := time.Now()
	if err := lt.createNamespacesBatch(ctx, startNS, endNS); err != nil {
		return fmt.Errorf("failed to create namespaces: %w", err)
	}
	fmt.Printf("  ✅ Created %d namespaces in %v\n", batchSize, time.Since(start))

	// Phase 2: Create pods for this batch (with delay)
	fmt.Printf("  Phase 2: Creating %d pods...\n", batchSize*lt.config.PodsPerNamespace)
	time.Sleep(1 * time.Second) // Small delay
	start = time.Now()
	if err := lt.createPodsBatch(ctx, startNS, endNS); err != nil {
		return fmt.Errorf("failed to create pods: %w", err)
	}
	totalPods := batchSize * lt.config.PodsPerNamespace
	fmt.Printf("  ✅ Created %d pods in %v\n", totalPods, time.Since(start))

	// Phase 3: Wait for live duration
	fmt.Printf("  Phase 3: Waiting %v for resources to live...\n", lt.config.LiveDuration)
	time.Sleep(lt.config.LiveDuration)

	// Phase 4: Cleanup this batch
	fmt.Printf("  Phase 4: Cleaning up batch...\n")
	start = time.Now()
	if err := lt.cleanupBatch(ctx, startNS, endNS); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}
	fmt.Printf("  ✅ Cleaned up batch in %v\n", time.Since(start))

	return nil
}

func (lt *LoadTester) createPodsWithDelay(ctx context.Context) error {
	// Small delay to let API server catch up and reduce throttling
	fmt.Println("Waiting 2s before creating pods...")
	time.Sleep(2 * time.Second)

	return lt.createPodsAll(ctx)
}

// executeInParallel is a generic helper for parallel task execution
func (lt *LoadTester) executeInParallel(tasks []func() error) error {
	if len(tasks) == 0 {
		return nil
	}

	manager := parallel.New(lt.config.MaxConcurrency)
	waiter := parallel.NewWaiter()

	for _, task := range tasks {
		manager.Run(task, waiter)
	}

	manager.Close()
	waiter.Wait()

	// Check for errors
	for err := range waiter.Err() {
		if err != nil {
			return err
		}
	}

	return nil
}

func (lt *LoadTester) createNamespacesBatch(ctx context.Context, startNS, endNS int) error {
	batchSize := endNS - startNS + 1
	tasks := make([]func() error, batchSize)

	for i := 0; i < batchSize; i++ {
		nsIndex := startNS + i // Capture value, not reference
		tasks[i] = func() error {
			ns := lt.buildNamespace(nsIndex)
			_, err := lt.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			return err
		}
	}

	return lt.executeInParallel(tasks)
}

func (lt *LoadTester) createNamespacesAll(ctx context.Context) error {
	tasks := make([]func() error, lt.config.NamespaceCount)

	for i := 0; i < lt.config.NamespaceCount; i++ {
		nsIndex := i + 1 // Capture value, not reference
		tasks[i] = func() error {
			ns := lt.buildNamespace(nsIndex)
			_, err := lt.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			return err
		}
	}

	return lt.executeInParallel(tasks)
}

func (lt *LoadTester) createPodsBatch(ctx context.Context, startNS, endNS int) error {
	batchSize := endNS - startNS + 1
	totalPods := batchSize * lt.config.PodsPerNamespace
	tasks := make([]func() error, totalPods)
	taskIndex := 0

	for nsIndex := startNS; nsIndex <= endNS; nsIndex++ {
		for podIndex := 1; podIndex <= lt.config.PodsPerNamespace; podIndex++ {
			// Capture values, not references
			ns, pod := nsIndex, podIndex

			tasks[taskIndex] = func() error {
				podObj := lt.buildPod(ns, pod)
				_, err := lt.client.CoreV1().Pods(podObj.Namespace).Create(ctx, podObj, metav1.CreateOptions{})
				return err
			}
			taskIndex++
		}
	}

	return lt.executeInParallel(tasks)
}

func (lt *LoadTester) createPodsAll(ctx context.Context) error {
	totalPods := lt.config.NamespaceCount * lt.config.PodsPerNamespace
	tasks := make([]func() error, totalPods)
	taskIndex := 0

	for nsIndex := 1; nsIndex <= lt.config.NamespaceCount; nsIndex++ {
		for podIndex := 1; podIndex <= lt.config.PodsPerNamespace; podIndex++ {
			// Capture values, not references
			ns, pod := nsIndex, podIndex

			tasks[taskIndex] = func() error {
				podObj := lt.buildPod(ns, pod)
				_, err := lt.client.CoreV1().Pods(podObj.Namespace).Create(ctx, podObj, metav1.CreateOptions{})
				return err
			}
			taskIndex++
		}
	}

	return lt.executeInParallel(tasks)
}

func (lt *LoadTester) buildNamespace(index int) *corev1.Namespace {
	labels := make(map[string]string)
	for k, v := range standardLabels {
		labels[k] = v
	}
	labels["nsgroup"] = lt.config.NSGroup
	labels["ns-index"] = fmt.Sprintf("%d", index)

	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%d", lt.config.NamespacePrefix, index),
			Labels: labels,
		},
	}
}

func (lt *LoadTester) buildPod(nsIndex, podIndex int) *corev1.Pod {
	labels := make(map[string]string)
	for k, v := range standardLabels {
		labels[k] = v
	}
	labels["app"] = "webhook-load-test"
	labels["nsgroup"] = lt.config.NSGroup
	labels["ns-index"] = fmt.Sprintf("%d", nsIndex)
	labels["pod-index"] = fmt.Sprintf("%d", podIndex)
	labels["resource-profile"] = lt.config.ResourceProfile

	// Build resource requirements from config
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(lt.config.MemoryRequest),
			corev1.ResourceCPU:    resource.MustParse(lt.config.CPURequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(lt.config.MemoryLimit),
			corev1.ResourceCPU:    resource.MustParse(lt.config.CPULimit),
		},
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", lt.config.PodPrefix, podIndex),
			Namespace: fmt.Sprintf("%s-%d", lt.config.NamespacePrefix, nsIndex),
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:      "nginx",
					Image:     "nginx:1.21-alpine",
					Ports:     []corev1.ContainerPort{{ContainerPort: 80}},
					Resources: resources,
					Env: []corev1.EnvVar{
						{Name: "NAMESPACE", Value: fmt.Sprintf("%s-%d", lt.config.NamespacePrefix, nsIndex)},
						{Name: "POD_INDEX", Value: fmt.Sprintf("%d", podIndex)},
						{Name: "NSGROUP", Value: lt.config.NSGroup},
						{Name: "RESOURCE_PROFILE", Value: lt.config.ResourceProfile},
					},
				},
			},
		},
	}
}

func (lt *LoadTester) validateResources(ctx context.Context) error {
	// Validate all expected namespaces exist
	namespaces, err := lt.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("nsgroup=%s", lt.config.NSGroup),
	})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	if len(namespaces.Items) != lt.config.NamespaceCount {
		return fmt.Errorf("expected %d namespaces, found %d", lt.config.NamespaceCount, len(namespaces.Items))
	}

	// Validate expected namespace names
	expectedNamespaces := make(map[string]bool)
	for i := 1; i <= lt.config.NamespaceCount; i++ {
		expectedNamespaces[fmt.Sprintf("%s-%d", lt.config.NamespacePrefix, i)] = false
	}

	for _, ns := range namespaces.Items {
		if _, expected := expectedNamespaces[ns.Name]; expected {
			expectedNamespaces[ns.Name] = true
		} else {
			return fmt.Errorf("unexpected namespace found: %s", ns.Name)
		}
	}

	for name, found := range expectedNamespaces {
		if !found {
			return fmt.Errorf("expected namespace not found: %s", name)
		}
	}

	// Validate all expected pods exist
	pods, err := lt.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("nsgroup=%s", lt.config.NSGroup),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	expectedTotalPods := lt.config.NamespaceCount * lt.config.PodsPerNamespace
	if len(pods.Items) != expectedTotalPods {
		return fmt.Errorf("expected %d pods, found %d", expectedTotalPods, len(pods.Items))
	}

	fmt.Printf("  - Found %d/%d expected namespaces\n", len(namespaces.Items), lt.config.NamespaceCount)
	fmt.Printf("  - Found %d/%d expected pods\n", len(pods.Items), expectedTotalPods)

	return nil
}

func (lt *LoadTester) cleanupBatch(ctx context.Context, startNS, endNS int) error {
	// Build expected namespace names for this batch
	var namesToDelete []string
	for nsIndex := startNS; nsIndex <= endNS; nsIndex++ {
		namesToDelete = append(namesToDelete, fmt.Sprintf("%s-%d", lt.config.NamespacePrefix, nsIndex))
	}

	// Delete namespaces in parallel (cascades to pods)
	deletePolicy := metav1.DeletePropagationForeground
	tasks := make([]func() error, len(namesToDelete))

	for i, nsName := range namesToDelete {
		name := nsName // Capture value
		tasks[i] = func() error {
			return lt.client.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
		}
	}

	return lt.executeInParallel(tasks)
}

func (lt *LoadTester) cleanupAll(ctx context.Context) error {
	// Get all namespaces with the nsgroup label
	namespaces, err := lt.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("nsgroup=%s", lt.config.NSGroup),
	})
	if err != nil {
		return fmt.Errorf("failed to list namespaces for cleanup: %w", err)
	}

	if len(namespaces.Items) == 0 {
		return nil // Nothing to clean up
	}

	// Delete all at once
	deletePolicy := metav1.DeletePropagationForeground
	tasks := make([]func() error, len(namespaces.Items))

	for i, ns := range namespaces.Items {
		nsName := ns.Name // Capture value
		tasks[i] = func() error {
			return lt.client.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
		}
	}

	return lt.executeInParallel(tasks)
}
