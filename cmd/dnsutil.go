package main

type Coredns struct {
	clusterIP         string
	endpointsIP       []string
	notReadyEndpoints []string
	namespace         string
	imageVersion      string
	recommVersion     string
	metrics           []string
	replicas          int
	corefile          []string
}
