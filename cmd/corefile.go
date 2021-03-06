package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/caddyserver/caddy/caddyfile"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getCorefile(ns string) (string, error) {
	api := Clientset.CoreV1()

	cm, err := api.ConfigMaps(ns).Get("coredns", metav1.GetOptions{})
	if err != nil {
		log.Errorf("coredns configmap does not exist %s", err)
		return "", err
	}
	corefile := cm.Data["Corefile"]
	//log.Debugf("Corefile content is: %s", corefile)
	return corefile, nil
}

func parseCorefile(corefile string) (bool, error) {
	status := strings.Contains(corefile, "log")
	if status {
		log.Debugf("Log plugin is enabled: %t", status)
	} else {
		log.Debugf("Log plugin is NOT enabled: %t", status)
	}

	//write corefile to a temp file
	err := ioutil.WriteFile("/tmp/corefile", []byte(corefile), 0644)
	if err != nil {
		log.Errorf("Failed to write Corefile")
		return false, err
	}

	//currenty caddyfile is just parsing the corefile content and not doing anything else
	//In future, can be used for understanding/parsing Corefile fields and decisions based on them
	//Use caddyfile to parse corefile
	serverBlocks, err := caddyfile.Parse("/tmp/corefile", bytes.NewReader([]byte(corefile)), nil)
	if err != nil {
		log.Errorf("Failed to read/parse Corefile")
		return false, err
	}
	srvBlocks, err := json.Marshal(serverBlocks)
	log.Infof("ServerBlocks: %v %v", string(srvBlocks), err)

	//trim corefile and obtain .:53 serverblock with regexp
	re := regexp.MustCompile(`(?m)(^\.:53) {([\s\w\W]+(?::53)|[\s\w\W]+)`)
	found := re.FindAllString(corefile, -1)
	fmt.Printf("found=%v\n", found)
	//found[0]

	coreblock := corefile[:strings.LastIndexByte(corefile, '}')+1]
	log.Infof("Coreblock: %v", coreblock)
	return status, nil
}

//checkLogs - Check for Errors in the DNS pod  -> fetch logs of coredns pod
func checkLogs(podNames []string) (map[string]interface{}, error) {
	//example: for p in $(kubectl get pods --namespace=kube-system -l k8s-app=kube-dns -o name); do kubectl logs --namespace=kube-system $p; done
	//0. List all the pods running with kube-dns label in kube-system namespace
	//https://127.0.0.1:32768/api/v1/namespaces/kube-system/pods?labelSelector=k8s-app%3Dkube-dns&limit=500
	//kubectl logs -n kube-system --selector 'k8s-app=kube-dns' -> api/v1/namespaces/kube-system/pods?labelSelector=k8s-app=kube-dns

	//1. Get pods logs
	api := Clientset.CoreV1()
	req := api.Pods("kube-system").GetLogs(podNames[0], &v1.PodLogOptions{})

	log.Debugf("pod request object: %v", req)

	podLogs, err := req.Stream()
	if err != nil {
		log.Errorf("error in opening stream: %v", err)
		return nil, fmt.Errorf("error in opening stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, fmt.Errorf("error in copy information from podLogs to buf")
	}
	logContent := buf.String()
	//log.Debugf("Pod logs are %v", logContent)

	//2. Check if seeing any errors in the logs
	logResult := make(map[string]interface{})
	re := regexp.MustCompile(`error|timeout|unreachable`)
	errChecksInLogs := re.FindAllString(logContent, -1)
	if len(errChecksInLogs) != 0 {
		log.Debugf("Seeing errors in coredns logs")
		logResult["errors"] = errChecksInLogs
		logResult["errorsInLogs"] = true
	} else {
		log.Debugf("NO errors in coredns pod logs")
		logResult["errorsInLogs"] = false
	}
	//todo: check if DNS queries are being received/processed by coredns

	return logResult, err
}

func checkForErrorsInLogs(ns string, cd *Coredns) error {
	//1. Fetch the Corefile content
	log.Infof("Retrieving Corefile from the coredns configmap...")
	corefile, err := getCorefile(ns)
	if err != nil {
		log.Errorf("Failed to retrieve coredns configmap: %s", err)
		return fmt.Errorf("Failed to retrieve coredns configmap: %s", err)
	}
	log.Infof("Corefile content is %s", corefile)
	cd.Corefile = corefile

	//2. Parse Corefile content to check if log plugin is enabled or not
	isLogPluginEnabled, err := parseCorefile(corefile)
	if err != nil {
		log.Errorf("Failed to parse corefile: %s", err)
		return fmt.Errorf("Failed to parse corefile: %s", err)
	}

	//3. If log plugin is not enabled, enable it by updating/patching Configmap
	if !isLogPluginEnabled {
		log.Infof("Log Plugin is not enabled, skipping coredns logs checking...")
		return nil
	} else {
		//If log plugin is already enabled, check the coredns pod logs for:
		//1. Any errors
		//todo: 2. DNS queries are being receieved/processed or not
		errChecksInLogs, err := checkLogs(cd.PodNamesList)
		if err != nil {
			log.Errorf("Failed to check errors in logs: %v", err)
			return fmt.Errorf("Failed to check errors in logs: %v", err)
		}
		cd.ErrorsInCorednsLogs = errChecksInLogs
		log.Infof("Pod log retireval status: %v", errChecksInLogs)
	}
	return nil
}
