package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/caddyserver/caddy/caddyfile"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// enableLogPlugin Enables logging for DNS query debugging
// Adds log plugin in coredns configmap
func enableLogPlugin(ns string, corefile string) (bool, error) {
	//PATCH request

	api := Clientset.CoreV1()

	ind := strings.IndexByte(corefile, '{')
	log.Debugf("Index: %v", ind)

	patch := corefile[:ind+1] + "\n    log" + corefile[ind+1:]
	log.Infof("patch payload: %v", patch)
	patchedConfigmap, err := api.ConfigMaps(ns).Patch("coredns", types.StrategicMergePatchType, []byte(patch), "")
	if err != nil {
		log.Errorf("Failed to patch coredns configmap")
		return false, err
	}
	log.Infof("Successfully patched coredns configmap : %v", patchedConfigmap)
	return true, nil
}

func checkLogs() {

}

func checkForErrorsInLogs(ns string, cd *Coredns) (string, error) {

	//1. Fetch the Corefile content
	log.Infof("Retrieving Corefile from the coredns configmap...")
	corefile, err := getCorefile(ns)
	if err != nil {
		log.Errorf("Failed to retrieve coredns configmap: %s", err)
		return "", err
	}
	log.Infof("Corefile content is %s", corefile)
	cd.corefile = corefile

	//2. Parse Corefile content to check if log plugin is enabled or not
	isLogPluginEnabled, err := parseCorefile(corefile)
	if err != nil {
		log.Errorf("Failed to parse corefile: %s", err)
		return "", err
	}

	//3. If log plugin is not enabled, enable it by updating/patching Configmap
	if !isLogPluginEnabled {
		result, err := enableLogPlugin(ns, corefile)
		if err != nil {
			log.Errorf("Failed to enable log plugin in coredns configmap: %v", err)
			return "", err
		}
		log.Infof("updated configmap: %v", result)
	} else {
		//If log plugin is already enabled, check the coredns pod logs for:
		//1. Any errors
		//2. DNS queries are being receieved/processed or not
		checkLogs()
	}

	return "eee", err
}
