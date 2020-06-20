package main

import (
	"fmt"
	"github.com/joshisumit/eks-dns-troubleshooter/version"
	log "github.com/sirupsen/logrus"
	"os"
)

//const summaryFilePath = "/var/log/eks-dns-tool.log"
const summaryFilePath = "/var/log/eks-dns-diag-summary.log"

type diagnosisSummary struct {
	isDiagComplete      bool
	isDiagSuccessful    bool
	diagError           string
	kubeDnsServiceExist map[string]interface{}
	corednsEndpoints    map[string]interface{}
	dnsResoution        map[string]interface{}
	recommendedVersion  bool
	envDetails          map[string]interface{}
	secGroupChecks      map[string]interface{}
	naclChecks          map[string]interface{}
}

//sample Output:
/*
DNS Resolution: [OK]


*/

func (ds diagnosisSummary) printSummary() {
	//Generate the file
	f, err := os.Create(summaryFilePath)
	if err != nil {
		log.Fatalf("Failed to create summary file: %v, ", err)
	}

	data := make([]string, 0)
	data = append(data, version.ShowVersion())
	data = append(data, "DNS Dignosis Summary: ")

	data = append(data, fmt.Sprintf("Diagnosis Status: %v", ds.isDiagComplete))

	data = append(data, fmt.Sprintf("Diagnosis Error: %v", ds.diagError))
	data = append(data, fmt.Sprintf("Kudedns service: %v", ds.kubeDnsServiceExist))
	data = append(data, fmt.Sprintf("coredns endpoints: %v", ds.corednsEndpoints))
	data = append(data, fmt.Sprintf("Kudedns service: %v", ds.corednsEndpoints))

	data = append(data, "isDiagSuccessful: ")

	for _, v := range data {
		_, err = fmt.Fprintln(f, v)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("file written successfully")

}
