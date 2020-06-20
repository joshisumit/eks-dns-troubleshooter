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
	IsDiagComplete     bool   `json:"diagnosisCompletion"`
	IsDiagSuccessful   bool   `json:"diagnosisResult"`
	DiagError          string `json:"diagnosisError"`
	EksVersion         string
	Release            string
	Repo               string
	Commit             string
	Coredns            `json:"corednsInfo"`
	EksClusterChecks   aws.ClusterInfo `json:"eksClusterChecks"`
	RecommendedVersion bool
}

func (ds DiagnosisSummary) printSummary() {
	//Generate the file
	fmt.Println("Printing summary....")

	// Create JSON Marshal
	report, err := json.Marshal(ds)
	if err != nil {
		log.Errorf("Failed to Marshal: %v", err)
	}

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
