package main

import (
	log "github.com/sirupsen/logrus"
	"net"

	"github.com/bogdanovich/dns_resolver"
)

//Coredns struct sets all the properties of coredns
type Coredns struct {
	ClusterIP         string     `json:"clusterIP"`
	EndpointsIP       []string   `json:"endpointsIP"`
	NotReadyEndpoints []string   `json:"notReadyEndpoints"`
	Namespace         string     `json:"namespace"`
	ImageVersion      string     `json:"imageVersion"`
	RecommVersion     string     `json:"recommendedVersion"`
	Dnstest           Dnstest    `json:"dnstestResults"`
	Metrics           []string   `json:"metrics,omitempty"`
	Replicas          int32      `json:"replicas"`
	PodNamesList      []string   `json:"podNames"`
	Corefile          string     `json:"corefile"`
	ResolvConf        ResolvConf `json:"resolvconf"`
	HasNodeLocalCache bool       `json:"isNodeLocalCacheEnabled,omitempty"`
	//nodeLocalCacheIP  string -> should be set manually to 169.254.20.10
	ErrorsInCorednsLogs map[string]interface{} `json:"errorCheckInCorednsLogs,omitempty"`
}

type DnsTestResultForDomain struct {
	DomainName string   `json:"domain"`
	Server     string   `json:"server"`
	Result     string   `json:"result"`
	Answer     []string `json:"answer,omitempty"`
}

type Dnstest struct {
	DnsResolution           string                   `json:"dnsResolution"`
	Description             string                   `json:"description,omitempty"`
	DomainsTested           []string                 `json:"domainsTested,omitempty"`
	DnsTestResultForDomains []DnsTestResultForDomain `json:"detailedResultForEachDomain,omitempty"`
}

func lookupIP(host string, server []string) *DnsTestResultForDomain {
	var (
		result string
		s, f   int
		ip     []net.IP
	)

	srv := server // creating a local copy
	testres := DnsTestResultForDomain{}

	resolver := dns_resolver.New(srv)

	// In case of io timeout, retry 3 times
	resolver.RetryTimes = 3

	//Perform each DNS query for 3 times
	answer := make([]string, 0)
	for i := 1; i <= 3; i++ {
		log.Infof("DNS query: %s Servers: %v", host, srv)
		ip, err := resolver.LookupHost(host)
		if err != nil {
			log.Errorf("Failed to resolve DNS query: %v %v ==> %s", host, ip, err.Error())
			f++
			continue
		}
		s++
	}

	for _, ipaddr := range ip {
		answer = append(answer, ipaddr.String())
	}
	log.Infof("Answer: %s A %s %v", host, ip, answer)

	log.Debugf("success: %d fail: %d domain: %s Servers: %s", s, f, host, srv)
	if f > 0 {
		log.Errorf("DNS query failed %d times", f)
		result = "failed"
	} else {
		log.Infof("DNS queries succeeded %d times", s)
		result = "success"
	}

	testres.DomainName, testres.Server, testres.Result, testres.Answer = host, srv[0], result, answer

	return &testres
}
