package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// func clusterVersion() (string, error) {
// 	return Clientset.ServerVersion()
// }

func getClusterIP(ns string) (string, error) {
	api := Clientset.CoreV1()

	getOptions := metav1.GetOptions{}

	svc, err := api.Services(ns).Get("kube-dns", getOptions)
	if err != nil {
		log.Errorf("kube-dns service does not exist %s", err)
		return "", err
		//redirect to central suggestion function
	}
	clusterIP := svc.Spec.ClusterIP
	return clusterIP, err
}

func checkServieEndpoint(ns string) ([]string, []string, error) {
	api := Clientset.CoreV1()

	endpoints, err := api.Endpoints(ns).Get("kube-dns", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("kube-dns endpoints does not exist %s", err)
		return nil, nil, err
		//redirect to central suggestion function
	}

	eips := make([]string, 0)
	notReadyEIP := make([]string, 0)

	log.Infof("Endpoints addresses: %v", endpoints.Subsets[0])

	for _, addr := range endpoints.Subsets[0].Addresses {
		log.Infof("Endpoints endpoints.Subsets[0].Addresses: %v", endpoints.Subsets[0].Addresses)
		eips = append(eips, addr.IP)
	}
	log.Infof("kube-dns endpoint IPs: %v length: %d", eips, len(eips))
	for _, addr := range endpoints.Subsets[0].NotReadyAddresses {
		log.Infof("Endpoints endpoints.Subsets[0].NotReadyAddresses: %v", endpoints.Subsets[0].NotReadyAddresses)
		notReadyEIP = append(notReadyEIP, addr.IP)
		log.Infof("Coredns pod IPs which are not ready: %s", addr.IP)
		//redirect to main...where inform that multiple Coredns pods are in notReady status
	}
	log.Infof("kube-dns notReadyEIPs: %v length: %d", notReadyEIP, len(notReadyEIP))
	return eips, notReadyEIP, err
}

func checkPodVersion(ns string, cd *Coredns) (string, error) {

	//There are 2 replicas of coredns pods running:
	//podNames are: x1 y1

	getOptions := metav1.GetOptions{}

	dep, err := Clientset.AppsV1().Deployments(ns).Get("coredns", getOptions)
	if err != nil {
		log.Fatalf("Failed to check coredns deployment %s", err)
	}

	replicas := dep.Spec.Replicas
	log.Infof("There are %d replicas of coredns pods running:", replicas)

	img := dep.Spec.Template.Spec.Containers[0].Image
	image := strings.Split(img, ":")

	name, tag := image[0], image[1]
	cd.ImageVersion = tag

	log.Infof("Image version: %s %s", name, tag)
	return tag, err
}

//testDNS tests the DNS resolution for differnt domain names...Just a simple DNS resolver based on => github.com/miekg/dns
//It tests the DNS queries against ClusterIP and individual PodIPs (i.e endpoint IPs)
//returns (bool, []byte, error)
func (cd *Coredns) testDNS() {
	var success bool

	//1. readEtcResolvConf -> compare nameserver with ClusterIP
	//nameserver either should be coredns clusterIP or nodeLocalcache DNS IP
	rc := &ResolvConf{}
	err := rc.readResolvConf()
	if err != nil {
		log.Errorf("Failed to read /etc/resolv.conf file: %s", err)
		cd.Dnstest = false
		return
	}
	cd.ResolvConf = *rc
	log.Infof("resolvconf values are: %+v", rc)

	//2. Match nameserver in /etc/resolv.conf with ClusterIP ->it should match
	//from the nameserver IP -> check its coredns or nodeLocalDNSCache
	if rc.Nameserver[0] == cd.ClusterIP {
		log.Infof("Pod's nameserver is matching to ClusterIP: %s", rc.Nameserver[0])
	} else if rc.Nameserver[0] == "169.254.20.10" {
		cd.HasNodeLocalCache = true
		log.Infof("Pod's nameserver is matching to NodeLocal DNS Cache: %s", rc.Nameserver[0])
	} else {
		log.Warnf("Pod's Nameserver is not set to Coredns clusterIP or NodeLocal Cache IP...Review the --cluster-dns parameter of kubelet or check dnsPolicy field of Pod")
	}

	//3. Test the DNS queries against multiple domains and host
	//As per miekg/dns library, domain names MUST be fully qualified before sending them, unqualified names in a message will result in a packing failure.
	//Fqdn() just adds . at the end of the query
	//If you make query for "kuberenetes" then query will be sent to COREDNS as "kubernetes."
	//Due to that used FQDN for kubernetes like kubernetes.default.svc.cluster.local
	domains := []string{"amazon.com", "kubernetes.default.svc.cluster.local"}

	nameservers := make([]string, 0, 3)
	nameservers = append(nameservers, rc.Nameserver...)
	nameservers = append(nameservers, cd.EndpointsIP[:2]...) //select only 2 endpoints

	// queries := make(map[string][]string)

	// queries = map[string][]string{
	// 	"amazon.com":         nameservers,
	// 	"kubernetes.default": nameservers,
	// }

	// for dom, ns := range queries {
	// 	success = lookupIP(dom, []string{ns})
	// }

	//tests each DOMAIN against 3 NAMESERVERS (i.e. 1 ClusterIP and 2 COREDNS ENDPOINTS)
	for _, dom := range domains {
		for _, ns := range nameservers {
			success = lookupIP(dom, []string{ns})
			if success {
				log.Infof("DNS queries suceeded for domain: %s server: %v", dom, ns)
			} else {
				log.Errorf("DNS queries failed for domain: %s server: %v", dom, ns)
			}
		}
	}

	cd.Dnstest = success
	log.Debugf("DNS test completed. Success: %t %t", cd.Dnstest, success)

}
