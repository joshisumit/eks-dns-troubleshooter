package main

import (
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// func clusterVersion() (string, error) {
// 	return Clientset.ServerVersion()
// }

func getClusterIP(ns string) (string, error) {
	api := Clientset.CoreV1()

	getOptions := metav1.GetOptions{}

	svc, err := api.Services(ns).Get("kube-dns", getOptions)
	if err != nil {
		log.Errorf("kube-dns service does not exist %s", err)
		return "", err
		//redirect to central suggestion function
	}
	clusterIP := svc.Spec.ClusterIP
	return clusterIP, err
}

func checkServieEndpoint(ns string) ([]string, error) {
	api := Clientset.CoreV1()

	getOptions := metav1.GetOptions{}

	endpoints, err := api.Endpoints(ns).Get("kube-dns", getOptions)
	if err != nil {
		log.Fatalf("kube-dns endpoints does not exist %s", err)
		return nil, err
		//redirect to central suggestion function
	}

	eips := make([]string, 4)

	for _, e := range endpoints.Subsets {
		for _, addr := range e.Addresses {
			eips = append(eips, addr.IP)
		}
		//log.Infof("kube-dns endpoint IPs: %v", eips)
		for _, addr := range e.NotReadyAddresses {
			log.Infof("Coredns pod IPs which are not ready: %s", addr.IP)
			//redirect to main...where inform that multiple Coredns pods are in notReady status
		}
	}
	return eips, err
}

func checkPodVersion(ns string) (string, error) {

	//There are 2 replicas of coredns pods running:
	//podNames are: x1 y1

	getOptions := metav1.GetOptions{}

	dep, err := Clientset.AppsV1().Deployments(ns).Get("coredns", getOptions)
	if err != nil {
		log.Fatalf("Failed to check coredns deployment %s", err)
	}

	replicas := dep.Spec.Replicas
	log.Infof("There are %d replicas of coredns pods running:", replicas)

	img := dep.Spec.Template.Spec.Containers[0].Image
	log.Infof("Image version: %s", img)
	return img, err
}
