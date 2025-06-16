package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Recommendation struct {
	Namespace     string
	PodName       string
	ContainerName string
	CPURequest    int // in millicores
	CPULimit      int
	MemRequest    int64 // in bytes
	MemLimit      int64
}

func main() {
	clientset := getClientset()
	recs, err := loadRecommendations("./recs.json")
	if err != nil {
		log.Fatalf("Failed to load recommendations: %v", err)
	}

	for _, rec := range recs {
		err := patchPodResources(clientset, rec)
		if err != nil {
			log.Printf("❌ Failed to patch pod %s/%s: %v", rec.Namespace, rec.PodName, err)
		} else {
			log.Printf("✅ Patched %s/%s for container %s", rec.Namespace, rec.PodName, rec.ContainerName)
		}
		// Optional: sleep between updates to reduce API pressure
		time.Sleep(300 * time.Millisecond)
	}
}

func getClientset() *kubernetes.Clientset {
	var config *rest.Config
	var err error

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not create Kubernetes client: %v", err)
	}
	return clientset
}

func getRecommendationForPod(pod *corev1.Pod, recs map[string]Recommendation) (*Recommendation, error) {
	if len(pod.OwnerReferences) == 0 {
		return nil, fmt.Errorf("pod %s/%s has no owner", pod.Namespace, pod.Name)
	}

	owner := pod.OwnerReferences[0]
	if owner.Kind != "StatefulSet" && owner.Kind != "ReplicaSet" {
		return nil, fmt.Errorf("unsupported owner kind: %s", owner.Kind)
	}

	ownerName := owner.Name

	// If it's a ReplicaSet, get the Deployment name from the ReplicaSet name
	if owner.Kind == "ReplicaSet" {
		// Deployment creates ReplicaSet with name like: my-deployment-75cb66cbcf
		// You need to remove the hash suffix
		parts := strings.Split(ownerName, "-")
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed ReplicaSet name: %s", ownerName)
		}
		ownerName = strings.Join(parts[:len(parts)-1], "-")
	}

	key := ownerName + pod.Namespace

	rec, exists := recs[key]
	if !exists {
		return nil, fmt.Errorf("no recommendation found for key: %s", key)
	}

	return &rec, nil
}

func loadRecommendations(path string) ([]Recommendation, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var recs []Recommendation
	err = json.Unmarshal(data, &recs)
	return recs, err
}

func patchPodResources(clientset *kubernetes.Clientset, rec Recommendation) error {
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []map[string]interface{}{
				{
					"name": rec.ContainerName,
					"resources": map[string]interface{}{
						"requests": map[string]string{
							"cpu":    formatCPU(rec.CPURequest),
							"memory": formatMemory(rec.MemRequest),
						},
						"limits": map[string]string{
							"cpu":    formatCPU(rec.CPULimit),
							"memory": formatMemory(rec.MemLimit),
						},
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(rec.Namespace).Patch(
		context.Background(),
		rec.PodName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	return err
}
