package main

import (
	"encoding/json"
	"fmt"
	"github.com/joshisumit/eks-dns-troubleshooter/pkg/aws"
	"github.com/joshisumit/eks-dns-troubleshooter/version"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

const summaryFilePath = "/var/log/eks-dns-diag-summary.json"

// DiagnosisSummary delivers a JSON-formatted final diagnostic summary, written to a file.
// A complete example report that was generated on an uncaught error is provided in docs directory for reference.
type DiagnosisSummary struct {
	IsDiagComplete bool                 `json:"diagnosisCompletion"`
	DiagToolInfo   version.DiagToolInfo `json:"diagnosisToolInfo"`
	//IsDiagSuccessful bool                   `json:"diagnosisResult"`
	DiagError   string                 `json:"diagnosisError,omitempty"`
	Result      map[string]interface{} `json:"Analysis,omitempty"`
	EksVersion  string                 `json:"eksVersion"`
	Coredns     Coredns                `json:"corednsChecks"`
	ClusterInfo aws.ClusterInfo        `json:"eksClusterChecks"`
	//RecommendedVersion bool
}

// evalDiagResult evaluates diagnose result
func (ds *DiagnosisSummary) evalDiagResult() map[string]interface{} {
	//result:{"dnstest":"","awsSGChecks":"","awsNaclChecks":"","corednsInfo":"",}
	res := make(map[string]interface{})

	if ds.Coredns.Dnstest {
		res["dnstest"] = "DNS resolution is working correctly in the cluster"
	}
	if ds.ClusterInfo.NaclRulesCheck {
		res["naclRules"] = "naclRules are configured correctly...NOT blocking any DNS communication"
	} else {
		res["naclRules"] = "naclRules are NOT configured correctly...blocking DNS communication"
	}
	if ds.ClusterInfo.SgRulesCheck.IsClusterSGRuleCorrect {
		res["securityGroupConfigurations"] = "securityGroups are configured correctly...not blocking any DNS communication"
	} else {
		res["securityGroupConfigurations"] = ds.ClusterInfo.SgRulesCheck
	}

	return res
}

func (ds *DiagnosisSummary) printSummary() error {
	fmt.Println("Printing summary....")

	//1. evaulate final diagnosis and add Result field in the DiagnosisSummary struct
	resultAnalysis := ds.evalDiagResult()
	if len(resultAnalysis) != 0 {
		ds.Result = resultAnalysis
	}

	// 2. Create JSON Marshal
	fmt.Printf("Inside sum Type: %T value: %+v \n coredns struct: Type: %T value: %+v\n\n\n", ds, ds, ds.Coredns, ds.Coredns)
	report, err := json.Marshal(ds)
	if err != nil {
		log.Errorf("Failed to Marshal: %v", err)
		return fmt.Errorf("Failed to Marshal: %v", err)
	}

	//3. write JSON to file
	log.Printf("JSON formatted report output")
	fmt.Println(string(report))
	err = ioutil.WriteFile(summaryFilePath, report, 0644)
	if err != nil {
		log.Errorf("Failed to write to summary file: %v", err)
		return fmt.Errorf("Failed to write to summary file: %v", err)
	}
	fmt.Println("JSON file written successfully")

	return nil
}
