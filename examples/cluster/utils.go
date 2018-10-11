package main

import (
	"fmt"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/strvals"
)

func watchTillerUntilReady(namespace string, client kubernetes.Interface, timeout int64) bool {
	deadlinePollingChan := time.NewTimer(time.Duration(timeout) * time.Second).C
	checkTillerPodTicker := time.NewTicker(500 * time.Millisecond)
	doneChan := make(chan bool)

	defer checkTillerPodTicker.Stop()

	go func() {
		for range checkTillerPodTicker.C {
			_, err := portforwarder.GetTillerPodImage(client.CoreV1(), namespace)
			if err == nil {
				doneChan <- true
				break
			}
		}
	}()

	for {
		select {
		case <-deadlinePollingChan:
			return false
		case <-doneChan:
			return true
		}
	}
}

func waitforIngress(client kubernetes.Interface, namespace string, name string, timeout int64) bool {
	doneChan := make(chan bool)
	deadlineTimer := time.NewTimer(time.Duration(timeout) * time.Second).C
	ingressCheckTicker := time.NewTicker(15 * time.Second)
	defer ingressCheckTicker.Stop()

	go func() {
		for range ingressCheckTicker.C {
			fmt.Println("Checking ingress")
			ingResp, err := client.ExtensionsV1beta1().Ingresses(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Error getting ingress %v.%v: %v\n", name, namespace, err)
				continue
			}
			statusMap := make(map[string]string)
			annotation, ok := ingResp.ObjectMeta.Annotations["ingress.kubernetes.io/backends"]
			if !ok {
				fmt.Println("Waiting for ingress backends")
				continue
			}
			err = yaml.Unmarshal([]byte(annotation), &statusMap)
			if err != nil {
				fmt.Printf("Error parsing %v: %v\n", annotation, err)
				continue
			}

			wait := false
			for k, v := range statusMap {
				if v != "HEALTHY" {
					fmt.Printf("Backend %v is %v\n", k, v)
					wait = true
				}
			}
			if wait {
				continue
			}

			fmt.Println(statusMap)
			fmt.Println("Ingress status OK")
			doneChan <- true
			break
		}
	}()

	for {
		select {
		case <-deadlineTimer:
			return false
		case <-doneChan:
			return true
		}
	}
}

func waitForJobs(client kubernetes.Interface, namespace string, timeout int64) bool {
	doneChan := make(chan bool)
	deadlineTimer := time.NewTimer(time.Duration(timeout) * time.Second).C
	jobsCheckTicker := time.NewTicker(15 * time.Second)
	defer jobsCheckTicker.Stop()

	go func() {
		for range jobsCheckTicker.C {
			fmt.Println("Checking jobs for completion")
			jobsResp, err := client.BatchV1().Jobs(namespace).List(metav1.ListOptions{})
			if err != nil {
				fmt.Printf("Error listing jobs in %v: %v\n", namespace, err)
				continue
			}

			wait := false
			for _, job := range jobsResp.Items {
				if job.Status.CompletionTime.IsZero() {
					fmt.Printf("Waiting on job %v to finish\n", job.ObjectMeta.Name)
					wait = true
				}
			}
			if wait {
				continue
			}

			fmt.Println("All jobs complete")
			doneChan <- true
			break
		}
	}()

	for {
		select {
		case <-deadlineTimer:
			return false
		case <-doneChan:
			return true
		}
	}
}

func vals(valuesFiles []string, values []string) ([]byte, error) {
	base := map[string]interface{}{}

	for _, filePath := range valuesFiles {
		currentMap := map[string]interface{}{}
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
			return []byte{}, fmt.Errorf("failed to parse %s: %s", filePath, err)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}

	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing --set data: %s", err)
		}
	}

	return yaml.Marshal(base)
}

// Merges source and destination map, preferring values from the source map
func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}
