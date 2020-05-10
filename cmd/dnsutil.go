package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/bogdanovich/dns_resolver"
)

//Coredns struct sets all the properties of coredns
type Coredns struct {
	clusterIP         string
	endpointsIP       []string
	notReadyEndpoints []string
	namespace         string
	imageVersion      string
	recommVersion     string
	metrics           []string
	replicas          int
	corefile          string
	dnstest           bool
	resolvconf
	hasNodeLocalCache bool
	//nodeLocalCacheIP  string -> should be set manually to 169.254.20.10
	hasErrorsInLogs map[bool]string
}

func lookupIP(host string, server []string) bool {
	var (
		success bool
		s, f    int
	)

	srv := server // creating a local copy

	resolver := dns_resolver.New(srv)

	// In case of io timeout, retry 3 times
	resolver.RetryTimes = 3

	//Perform each DNS query for 3 times
	for i := 1; i <= 3; i++ {
		log.Infof("DNS query: %s Servers: %v", host, srv)
		ip, err := resolver.LookupHost(host)
		if err != nil {
			log.Errorf("Failed to resolve DNS query: %v %v ==> %s", host, ip, err.Error())
			f++
			continue
		}
		s++
		log.Infof("Answer: %s A %s", host, ip)
	}

	log.Debugf("success: %d fail: %d domain: %s Servers: %s", s, f, host, srv)
	if f > 0 {
		log.Errorf("DNS query failed %d times", f)
		success = false
	} else {
		log.Infof("DNS queries succeded %d times", s)
		success = true
	}

	return success
}
