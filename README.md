# EKS DNS troubleshooter

EKS DNS troubleshooter is an automated DNS troubleshooting utility for EKS cluster. It can be used to test, validate and troubleshoot DNS issues in EKS cluster.

This tool scans all the cluster configuration, performs DNS resolution and identifies the issue, then generates a diagnosis report in JSON format.

Spend less time in finding a root cause, more on Important Stuff!

## Scenarios
Tool verifies the following scenarios to validate/troubleshoot DNS in EKS cluster:
- Check if coredns pods are running and number of replicas
- Recommended version of coredns pods are running (i.e. v1.6.6 as of now).
- Verify coredns service (i.e. kube-dns) exist and its endpoints.
- Performs DNS resolution against CoreDNS ClusterIP (e.g. 10.100.0.10) and individual Coredns pod IPs.
- Detects if Node Local DNS cache is being used.
- Verify that the inter-node communication is not blocked by a Security Group.
- Verify EKS Cluster Security Group is configured correctly (Incorrect configs can prevent communication with coredns pods).
- Verify Network Access Control List (NACL) rules are not blocking outbound TCP and UDP access on port 53 (which is required for DNS resolution).
- Enable `log` plugin in Coredns Configmap for debugging and checks for errors in the Coredns pod logs

## Usage
Note: Tool supports EKS version 1.14 onwards
1. Create an IAM policy and attach it to the Worker node IAM role.

```
wget https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/master/deploy/iam-policy.json

aws iam put-role-policy --role-name $ROLE_NAME --policy-name EKSDnsDiagToolIAMPolicy --policy-document file://iam-policy.json
```

Here replace $ROLE_NAME with EKS worker node IAM role

2. Deploy eks-dns-troubleshooter
```
kubectl apply -f https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/master/deploy/eks-dns-troubleshooter.yaml
```

3. Verify that pod is deployed
```
kubectl logs -l=app=eks-dns-troubleshooter
```
Should display output similar to the following.
```
time="2020-06-27T15:22:34Z" level=info msg="\n\t-----------------------------------------------------------------------\n\tEKS DNS Troubleshooter\n\tRelease:    v1.0.0\n\tBuild:      git-bc217f7\n\tRepository: git@github.com:joshisumit/eks-dns-troubleshooter.git\n\t-----------------------------------------------------------------------\n\t"
{"file":"main.go:81","func":"main","level":"info","msg":"Running on Kubernetes v1.15.11-eks-af3caf","time":"2020-06-27T15:22:34Z"}
{"file":"main.go:100","func":"main","level":"info","msg":"kube-dns service ClusterIP: 10.100.0.10","time":"2020-06-27T15:22:34Z"}
{"file":"verify-configs.go:42","func":"checkServieEndpoint","level":"info","msg":"Endpoints addresses: {[{192.168.27.238  0xc0004488a0 \u0026ObjectReference{Kind:Pod,Namespace:kube-system,Name:coredns-6658f9f447-gm5sn,UID:75d7a6cf-4092-4501-a56f-d3c859cdc5d0,APIVersion:,ResourceVersion:36301646,FieldPath:,}} {192.168.5.208  0xc0004488b0 \u0026ObjectReference{Kind:Pod,Namespace:kube-system,Name:coredns-6658f9f447-gqqk2,UID:f1864697-db9d-460c-b219-d5fbbe7484b8,APIVersion:,ResourceVersion:36301561,FieldPath:,}}] [] [{dns 53 UDP} {dns-tcp 53 TCP}]}","time":"2020-06-27T15:22:34Z"}
```
Once pod is running it will validate DNS, takes 1-2 mins for generating diagnosis report.

4. Exec into pod and fetch the diagnosis report

``` 
POD_NAME=$(kubectl get pods -l=app=eks-dns-troubleshooter -o jsonpath='{.items..metadata.name}')
kubectl exec -ti $POD_NAME -- cat /var/log/eks-dns-diag-summary.log | jq
```
OR download the diagnosis report JSON file to your machine

```
kubectl exec -ti $POD_NAME -- cat /var/log/eks-dns-diag-summary.log > eks-dns-diag-summary.json
```

Sample diagnosis report in JSON format looks similar to the following:

```
{
    "diagnosisCompletion": true,
    "diagnosisToolInfo": {
      "release": "v1.0.0",
      "repo": "git@github.com:joshisumit/eks-dns-troubleshooter.git",
      "commit": "git-bc217f7"
    },
    "Analysis": {
      "dnstest": "DNS resolution is working correctly in the cluster",
      "naclRules": "naclRules are configured correctly...NOT blocking any DNS communication",
      "securityGroupConfigurations": "securityGroups are configured correctly...not blocking any DNS communication"
    },
    "eksVersion": "v1.15.11-eks-af3caf",
    "corednsChecks": {
      "clusterIP": "10.100.0.10",
      "endpointsIP": [
        "192.168.27.238",
        "192.168.5.208"
      ],
      "notReadyEndpoints": [],
      "namespace": "kube-system",
      "imageVersion": "v1.6.6",
      "recommendedVersion": "v1.6.6",
      "dnstestResults": true,
      "replicas": 0,
      "corefile": ".:53 {\n    log\n    errors\n    health\n    kubernetes cluster.local in-addr.arpa ip6.arpa {\n      pods insecure\n      upstream\n      fallthrough in-addr.arpa ip6.arpa\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
      "resolvconf": {
        "SearchPath": [
          "default.svc.cluster.local",
          "svc.cluster.local",
          "cluster.local",
          "eu-west-1.compute.internal"
        ],
        "Nameserver": [
          "169.254.20.10"
        ],
        "Options": [
          "ndots:5"
        ],
        "Ndots": 5
      },
      "hasNodeLocalCache": true
    },
    "eksClusterChecks": {
      "securityGroupChecks": {
        "IsClusterSGRuleCorrect": true,
        "InboundRule": {
          "isValid": "true"
        },
        "OutboundRule": {
          "isValid": "true"
        }
      },
      "naclRulesCheck": true,
      "region": "eu-west-1",
      "securityGroupIds": [
        "eksctl-ekstest-cluster-ClusterSharedNodeSecurityGroup-1IZCQSZ7P0UXK",
        "eksctl-ekstest-nodegroup-nodelocal-ng-SG-YC8V65Q46LJX"
      ],
      "clusterName": "ekstest",
      "clusterSecurityGroup": "sg-0c5a36ce2a6d9478e",
      "instanceIdentityDocument": {
        "devpayProductCodes": null,
        "marketplaceProductCodes": null,
        "availabilityZone": "eu-west-1c",
        "privateIp": "192.168.94.187",
        "version": "2017-09-30 ",
        "region": "eu-west-1",
        "instanceId": "i-08cec7471234560b5",
        "billingProducts": null,
        "instanceType": "t3.large",
        "accountId": "123456789012",
        "pendingTime": "0001-01-01T00:00:00Z",
        "imageId": "ami-0f9e9442edcd2faa2 ",
        "kernelId": "",
        "ramdiskId": "",
        "architecture": "x86_64"
      },
      "clusterDetails": {
        "Arn": "arn:aws:eks:eu-west-1:123456789012:cluster/ekstest",
        "CertificateAuthority": {
          "Data": "LS0tLS1CRUdJTiBDTT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
        },
        "ClientRequestToken": null,
        "CreatedAt": "2019-12-16T19:16:56Z",
        "EncryptionConfig": null,
        "Endpoint": "https://6D9A445312345678901234.sk1.eu-west-1.eks.amazonaws.com",
        "Identity": {
          "Oidc": {
            "Issuer": "https://oidc.eks.eu-west-1.amazonaws.com/id/6D9A445312345678901234"
          }
        },
        "Logging": {
          "ClusterLogging": [
            {
              "Enabled": true,
              "Types": [
                "api",
                "audit",
                "scheduler"
              ]
            },
            {
              "Enabled": false,
              "Types": [
                "authenticator",
                "controllerManager"
              ]
            }
          ]
        },
        "Name": "ekstest",
        "PlatformVersion": "eks.2",
        "ResourcesVpcConfig": {
          "ClusterSecurityGroupId": "sg-0c5a36ce2a6d9478e",
          "EndpointPrivateAccess": true,
          "EndpointPublicAccess": true,
          "PublicAccessCidrs": [
            "0.0.0.0/0"
          ],
          "SecurityGroupIds": [
            "sg-09f1a4e5b0354f30a"
          ],
          "SubnetIds": [
            "subnet-00de54436e27d1aca",
            "subnet-079dd819d880c94e9",
            "subnet-0702e6a0bf3755bd0",
            "subnet-0cdacba7c6aedbf9f",
            "subnet-0cf4f0cd18486953b",
            "subnet-0fef11a06f26811d4"
          ],
          "VpcId": "vpc-053567008548ceb24"
        },
        "RoleArn": "arn:aws:iam::123456789012:role/eksctl-ekstest-cluster-ServiceRole-7EJONRZBLZH0",
        "Status": "ACTIVE",
        "Tags": {
          "EKS-Cluster-Name": "ekstest"
        },
        "Version": "1.15"
      }
    }
  }
```




## Contribute
- More scenarios for DNS troubleshooting would be welcome.
- Any tips/PR on how to make the code cleaner or more idiomatic would be welcome.


