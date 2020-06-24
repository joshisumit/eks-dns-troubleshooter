package main

import (
	"encoding/json"
	"fmt"
	"github.com/joshisumit/eks-dns-troubleshooter/pkg/aws"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

//const summaryFilePath = "/var/log/eks-dns-tool.log"
const summaryFilePath = "/var/log/eks-dns-diag-summary.log"

//DiagnosisSummary delivers a JSON-formatted diagnostic summary, written to a file.
//todo: A complete example report that was generated on an uncaught error is provided below for reference.

type DiagnosisSummary struct {
	IsDiagComplete     bool            `json:"diagnosisCompletion"`
	IsDiagSuccessful   bool            `json:"diagnosisResult"`
	DiagError          string          `json:"diagnosisError"`
	EksVersion         string          `json:"eksVersion"`
	Release            string          `json:"release"`
	Repo               string          `json:"repo"`
	Commit             string          `json:"commit"`
	Coredns            Coredns         `json:"corednsInfo"`
	ClusterInfo        aws.ClusterInfo `json:"eksClusterChecks"`
	RecommendedVersion bool
}

func (ds *DiagnosisSummary) printSummary() {
	//Generate the file
	fmt.Println("Printing summary....")

	// sum:= DiagnosisSummary{IsDiagComplete:true, IsDiagSuccessful:true, DiagError:"", EksVersion:"v1.15.11-eks-af3caf", Release:"v1.0.0", Repo:"git@github.com:joshisumit/eks-dns-troubleshooter.git", Commit:"git-ac33df7", Coredns:{clusterIP:"10.100.0.10", endpointsIP:["192.168.27.238", "192.168.5.208"], notReadyEndpoints:[], namespace:"kube-system", imageVersion:"v1.6.6", recommVersion:"v1.6.6", metrics:[], replicas:0,
	// 	dnstest:true, resolvconf:{searchPath:["default.svc.cluster.local","svc.cluster.local","cluster.local","eu-west-1.compute.internal"], nameserver:["169.254.20.10"], options:[ndots:5], ndots:5}, hasNodeLocalCache:true, hasErrorsInLogs:map[]}, ClusterInfo:{region:, securityGroupIds:[eksctl-ekstest-cluster-ClusterSharedNodeSecurityGroup-1IZCQSZ7P0UXK eksctl-ekstest-nodegroup-nodelocal-ng-SG-YC8V65Q46LJX], clusterName:ekstest, tagList:[], clusterDetails:0xc0003f27e0, clusterSGID:sg-0c5a36ce2a6d9478e, sgRulesCheck:{isClusterSGRuleCorrect:true inboundRule:map[isValid:true] outboundRule:map[isValid:true]}, naclRulesCheck:true}, RecommendedVersion:true}

	// sum := DiagnosisSummary{
	// 	IsDiagComplete:     ds.IsDiagComplete,
	// 	IsDiagSuccessful:   ds.IsDiagSuccessful,
	// 	EksVersion:         ds.EksVersion,
	// 	Release:            ds.Release,
	// 	Repo:               ds.Repo,
	// 	Commit:             ds.Commit,
	// 	Coredns:            ds.Coredns,
	// 	ClusterInfo:        ds.ClusterInfo,
	// 	RecommendedVersion: ds.RecommendedVersion,
	// }

	// fmt.Printf("restructured struct: %v\n\n", sum)

	// Create JSON Marshal
	report, err := json.Marshal(ds)
	if err != nil {
		log.Errorf("Failed to Marshal: %v", err)
	}

	fmt.Printf("Inside sum Type: %T value: %+v \n coredns struct: Type: %T value: %+v\n\n\n", ds, ds, ds.Coredns, ds.Coredns)
	//write JSON to file
	log.Printf("JSON formatted report output")
	fmt.Println(string(report))
	err = ioutil.WriteFile(summaryFilePath, report, 0644)
	if err != nil {
		log.Errorf("Failed to write to summary file")
		return
	}
	fmt.Println("file written successfully")

}
