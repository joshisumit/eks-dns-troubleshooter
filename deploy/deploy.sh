#!/usr/bin/env bash

set -xe

## This shell script is used to deploy local code changes into Kind cluster (which is running locally)

VERSION=$1

kind get clusters

rm -rf ../cmd/app

GOOS=linux go build -o ./app ../cmd/

docker build -t eks-dns-troubleshooter:$VERSION .

kind load docker-image eks-dns-troubleshooter:$VERSION --name kind

#Update the pod/deployment manifest with the latest docker image
kubectl set image deployment/eks-dns-troubleshooter in-cluster=eks-dns-troubleshooter:$VERSION --record

#sed -i s/^      - image: .*$/      - image: eks-dns-troubleshooter:$VERSION/' deployment.yaml
#kubectl apply -f deployment.yaml

sleep 20
kubectl get pods


#kubectl logs Pod-name