package main

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const logFilePath = "/var/log/eks-dns-tool.log"
const version = "v0.1.0"

//Clientset will be used for accessing multiple k8s groups
var Clientset *kubernetes.Clientset

func main() {

	//0. Logging - write same logs to stdout and file simultaneously
	//Set Logging based on a file
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	mw := io.MultiWriter(os.Stdout, file)

	log.SetOutput(mw)
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)

	log.Infof("Starting EKS DNS Troubleshooter %s ...", version)

	//Create Clientset
	Clientset, err = CreateKubeClient()
	if err != nil {
		log.Fatalf("Failed to create clientset: %s", err)
	}

	//Detect cluster version
	srvVersion, err := Clientset.ServerVersion()
	if err != nil {
		log.Fatalf("Failed to fetch kubernetes version: %s", err)
	}
	log.Infof("Running on Kubernetes %s", srvVersion.GitVersion)

	//Check whether kube-dns service exist or not
	cd := Coredns{}
	var ns string
	ns = "kube-system"
	cd.namespace = ns

	clusterIP, err := getClusterIP(ns)
	if err != nil {
		log.Fatalf("kube-dns service does not exist %s", err)
		//redirect to central suggestion function
	}
	log.Infof("kube-dns service ClusterIP: %s", clusterIP)
	cd.clusterIP = clusterIP

	//Check endpoint exist or not
	eips, notReadyEIP, err := checkServieEndpoint(ns)
	if err != nil {
		log.Fatalf("kube-dns endpoints does not exist %s", err)
		//redirect to central suggestion function
	}
	cd.endpointsIP = eips
	cd.notReadyEndpoints = notReadyEIP
	log.Infof("kube-dns endpoint IPs: %v length: %d cd.endspointsIP: %v", eips, len(eips), cd.endpointsIP)
	for i, v := range cd.endpointsIP {
		log.Infof("Printing EIP value %d: %s", i, v)
	}

	//Check recommenedVersion of CoreDNS pod is running or not
	poVer, err := checkPodVersion(ns, &cd)
	cd.recommVersion = "v1.6.6"
	if err != nil {
		log.Fatalf("Failed to detect coredns Pod version %s", err)
	}
	if poVer == cd.recommVersion {
		log.Infof("Recommended coredns version % is running", poVer)
	} else {
		log.Infof("Current coredns pods are running older version %s ", poVer)
		log.Infof("Recommended version for EKS %s is %s", srvVersion.GitVersion, cd.recommVersion)
		//Suggest to Upgrade coredns version with latest image
	}

	// Test DNS resolution
	cd.testDNS()

	//checkForErrorsInLogs
	result, err := checkForErrorsInLogs(ns, &cd)
	fmt.Println(result)

	log.Infof("Printing struct %+v", cd)
	for {
		time.Sleep(1000)
	}
}

//CreateKubeClient returns ClientSet
func CreateKubeClient() (*kubernetes.Clientset, error) {
	//1. Connection- creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("Failed to create inClusterConfig: %s", err)
		return nil, err
	}

	//2. Create ClientSet
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to create clientset: %s", err)
		return nil, err
	}
	return Clientset, err
}
