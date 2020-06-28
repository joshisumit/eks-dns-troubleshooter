package main

import (
	"fmt"
	"github.com/joshisumit/eks-dns-troubleshooter/pkg/aws"
	"github.com/joshisumit/eks-dns-troubleshooter/version"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	//"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const logFilePath = "/var/log/eks-dns-tool.log"

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
	log.Infoln(version.ShowVersion())
	formatter := &log.JSONFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			s := strings.Split(f.Function, ".")
			funcName := s[len(s)-1]
			return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
		},
	}
	log.SetFormatter(formatter)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	//show version
	sum := DiagnosisSummary{}
	sum.DiagToolInfo.Release, sum.DiagToolInfo.Repo, sum.DiagToolInfo.Commit = version.RELEASE, version.REPO, version.COMMIT
	//log.Infof("Starting EKS DNS Troubleshooter %s ...", version)

	//Create Clientset
	Clientset, err = CreateKubeClient()
	if err != nil {
		log.Errorf("Failed to create clientset: %s", err)
		sum.DiagError = fmt.Sprintf("Failed to create clientset: %s", err)
		//sum.isDiagComplete = false
		//sum.isDiagSuccessful = false
		err = sum.printSummary()
		if err != nil {
			log.Errorf("Failed to printSummary: %v", err)
			//return or call other function
		}
	}

	//Detect cluster version
	srvVersion, err := Clientset.ServerVersion()
	if err != nil {
		log.Errorf("Failed to fetch kubernetes version: %s", err)
		sum.DiagError = fmt.Sprintf("Failed to fetch kubernetes version: %s", err)
		err = sum.printSummary()
		if err != nil {
			log.Errorf("Failed to printSummary: %v", err)
			//return or call other function
		}
	}
	sum.EksVersion = srvVersion.GitVersion
	log.Infof("Running on Kubernetes %s", srvVersion.GitVersion)

	//Check whether kube-dns service exist or not
	cd := Coredns{}
	var ns string
	ns = "kube-system"
	cd.Namespace = ns

	clusterIP, err := getClusterIP(ns)
	if err != nil {
		log.Errorf("kube-dns service does not exist %s", err)
		sum.DiagError = fmt.Sprintf("kube-dns service does not exist %s", err)
		err = sum.printSummary()
		if err != nil {
			log.Errorf("Failed to printSummary: %v", err)
			//return or call other function
		}
		//redirect to central suggestion function
	}
	log.Infof("kube-dns service ClusterIP: %s", clusterIP)
	cd.ClusterIP = clusterIP

	//Check endpoint exist or not
	eips, notReadyEIP, err := checkServieEndpoint(ns)
	if err != nil {
		log.Errorf("kube-dns endpoints does not exist %s", err)
		sum.DiagError = fmt.Sprintf("kube-dns endpoints does not exist %s", err)
		err = sum.printSummary()
		if err != nil {
			log.Errorf("Failed to printSummary: %v", err)
			//return or call other function
		}
		//redirect to central suggestion function
	}
	cd.EndpointsIP = eips
	cd.NotReadyEndpoints = notReadyEIP

	log.Infof("kube-dns endpoint IPs: %v length: %d cd.endspointsIP: %v", eips, len(eips), cd.EndpointsIP)

	//Check recommenedVersion of CoreDNS pod is running or not
	poVer, podNamesList, err := checkPodVersion(ns, &cd)
	cd.RecommVersion = "v1.6.6"
	cd.PodNamesList = podNamesList
	if err != nil {
		log.Errorf("Failed to detect coredns Pod version %s", err)
		sum.DiagError = fmt.Sprintf("Failed to detect coredns Pod version %s", err)
		err = sum.printSummary()
		if err != nil {
			log.Errorf("Failed to printSummary: %v", err)
			//return or call other function
		}
	}
	if poVer == cd.RecommVersion {
		log.Infof("Recommended coredns version %v is running", poVer)
		//sum.RecommendedVersion = true
	} else {
		log.Infof("Current coredns pods are running older version %s ", poVer)
		log.Infof("Recommended version for EKS %s is %s", srvVersion.GitVersion, cd.RecommVersion)
		//Suggest to Upgrade coredns version with latest image
	}
	//sum.Coredns = cd

	// Test DNS resolution
	cd.testDNS()
	//sum.Coredns = cd

	//checkForErrorsInLogs
	//todo: return values
	log.Infof("Checking logs of coredns pods for further debugging")
	err = checkForErrorsInLogs(ns, &cd)
	if err != nil {
		log.Errorf("Failed to check logs of coredns pods and enable log plugin. Reason: %v\n", err)
	}

	//copy content of coredns struct to sum struct
	sum.Coredns = cd

	clusterInfo := aws.DiscoverClusterInfo()
	sum.ClusterInfo = *clusterInfo
	log.Debugf("Printing clusterInfo struct %+v", clusterInfo)

	//sum.IsDiagSuccessful = true
	sum.IsDiagComplete = true

	log.Debugf("Printing coredns struct %+v", cd)
	log.Infof("Printing Final diagnosis summary...")
	err = sum.printSummary()
	if err != nil {
		log.Errorf("Failed to printSummary: %v", err)
		//return or call other function
	}

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
