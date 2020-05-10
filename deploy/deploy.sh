#!/usr/bin/env bash

set -xe

## This shell script is used to deploy local code changes into Kind cluster (which is running locally)

VERSION=$1

kind get clusters

rm -rf ../cmd/app

GOOS=linux go build -o ./app ../cmd/

docker build -t in-cluster:$VERSION .

kind load docker-image in-cluster:$VERSION --name kind

#Update the pod/deployment manifest with the latest docker image
kubectl set image deployment/goclient-test in-cluster=in-cluster:$VERSION --record

#sed -i s/^      - image: .*$/      - image: in-cluster:$VERSION/' deployment.yaml
#kubectl apply -f deployment.yaml

sleep 20
kubectl get pods


#kubectl logs Pod-name