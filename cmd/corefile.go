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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//  patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type corednsPatchValue struct {
	Data *corefileValue `json:"data"`
}

type corefileValue struct {
	Corefile string `json:"Corefile"`
}

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

	payl := corefile[:ind+1] + "\n    log" + corefile[ind+1:]
	log.Debugf("payl: %v", payl)

	//form JSON file
	patch := &corednsPatchValue{
		Data: &corefileValue{Corefile: payl},
	}

	//patch := fmt.Sprintf(`{"data":{"Corefile":"+%s+"}}`, payl)
	log.Infof("patch payload: %v", *patch)

	// payload := []patchStringValue{{
	// 	Op:    "replace",
	// 	Path:  "/data",
	// 	Value: patch,
	// }}
	// payloadBytes, _ := json.Marshal(payload)
	// log.Debugf("payloadBytes : %v", payloadBytes)

	patchedcm, err := json.Marshal(patch)
	log.Debugf("pactchedcm JSON before: %v", string(patchedcm))
	if err != nil {
		log.Errorf("Failed to parse the config")
		return false, err
	}

	//
	// corefileStanza, err := caddyfile.ToJSON([]byte(patch))
	// if err != nil {
	// 	log.Errorf("error: %v", err)
	// 	return false, err
	// }
	// log.Infof("corefileStanza: %v", string(corefileStanza))

	patchedConfigmap, err := api.ConfigMaps(ns).Patch("coredns", types.StrategicMergePatchType, patchedcm, "")
	//patchedConfigmap, err := api.ConfigMaps(ns).Patch("coredns", types.StrategicMergePatchType, patchedcm, "")
	//patchedConfigmap, err := api.ConfigMaps(ns).Patch("coredns", types.StrategicMergePatchType, []byte(patch), "")
	if err != nil {
		log.Errorf("Failed to patch coredns configmap: %v", patchedConfigmap)
		log.Infof("Pacthed configmap data: %v", patchedConfigmap.Data)
		return false, err
	}
	log.Infof("Successfully patched coredns configmap : %v", patchedConfigmap)
	return true, nil
}

func checkLogs() error {
	var err error
	//Goal: Check for Errors in the DNS pod  -> fetch logs of all the running coredns pod logs and check errors and see queries are being receieved
	//for p in $(kubectl get pods --namespace=kube-system -l k8s-app=kube-dns -o name); do kubectl logs --namespace=kube-system $p; done

	//1. List all the pods running with kube-dns label in kube-system namespace
	//https://127.0.0.1:32768/api/v1/namespaces/kube-system/pods?labelSelector=k8s-app%3Dkube-dns&limit=500

	//2. Get pods logs
	api := Clientset.CoreV1()
	req := api.Pods("kube-system").GetLogs("coredns-6955765f44-9mnlp", &v1.PodLogOptions{})

	log.Debugf("pod request object: %v", req)

	// podLogs, err := req.Stream()
	// if err != nil {
	// 	log.Errorf("error in opening stream")
	// }
	// defer podLogs.Close()

	// r := bufio.NewReader(podLogs)
	// for {
	// 	bytes, err := r.ReadBytes('\n')
	// 	if _, err := out.Write(bytes); err != nil {
	// 		return err
	// 	}

	// 	if err != nil {
	// 		if err != io.EOF {
	// 			return err
	// 		}
	// 		return nil
	// 	}
	// }

	//log.Debugf("Pod logs are %v", podLogs)
	return err
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
	cd.Corefile = corefile

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
		err = checkLogs()
		log.Infof("Pod log retireval status: %v", err)
	}

	return "eee", err
}
