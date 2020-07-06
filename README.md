# EKS DNS troubleshooter

[![Go Report](https://goreportcard.com/badge/github.com/joshisumit/eks-dns-troubleshooter)](https://goreportcard.com/report/github.com/joshisumit/eks-dns-troubleshooter) &nbsp;
![Docker Image CI](https://github.com/joshisumit/eks-dns-troubleshooter/workflows/Docker%20Image%20CI/badge.svg)

[Docs](https://joshisumit.github.io/eks-dns-troubleshooter/)

EKS DNS troubleshooter is an automated DNS troubleshooting utility for EKS cluster. It can be used to test, validate and troubleshoot DNS issues in EKS cluster.

This tool runs as a pod in EKS cluster and scans all the cluster configuration, performs DNS resolution and identifies the issue, then generates a diagnosis report in JSON format.

Spend less time in finding a root cause, more on Important Stuff!

## Scenarios
Tool verifies the following scenarios to validate/troubleshoot DNS in EKS cluster:
- Check if coredns pods are running and number of replicas
- Recommended version of coredns pods are running (e.g. `v1.6.6` as of now).
- Verify coredns service (i.e. `kube-dns`) exist and its endpoints.
- Performs DNS resolution against CoreDNS ClusterIP (e.g. `10.100.0.10`) and individual Coredns pod IPs.
- Detects if Node Local DNS cache is being used.
- Verify EKS Cluster Security Group is configured correctly (Incorrect configs can prevent communication with coredns pods).
- Verify Network Access Control List (NACL) rules are not blocking outbound TCP and UDP access on port 53 (which is required for DNS resolution).
- Checks for errors in the Coredns pod logs (Only If `log` plugin is enabled in Coredns Configmap).

## Usage

To deploy the EKS DNS Troubleshooter to an EKS cluster:
1. [Create an IAM OIDC provider](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html) and associate it with your cluster
2. Download an IAM policy for the EKS DNS troubleshooter pod that allows it to make calls to AWS APIs on your behalf. You can view the [policy document](https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/v1.1.0/deploy/iam-policy.json).

```bash
curl -o iam-policy.json https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/v1.1.0/deploy/iam-policy.json
```

3. Create an IAM policy called `DNSTroubleshooterIAMPolicy` using the policy downloaded in the previous step.

```bash
aws iam create-policy \
    --policy-name DNSTroubleshooterIAMPolicy \
    --policy-document file://iam-policy.json
```
Take note of the policy ARN that is returned.


4. Create a Kubernetes service account named `eks-dns-ts`, a cluster role, and a cluster role binding for the EKS DNS troubleshooter to use with the following command.

```bash
kubectl apply -f https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/v1.1.0/deploy/rbac-role.yaml
```

5. Create an IAM role for the EKS DNS Troubleshooter and attach the role to the service account created in the previous step.
   1. Create an IAM role named `eks-dns-troubleshooter` and attach the `DNSTroubleshooterIAMPolicy` IAM policy that you created in a previous step to it. Note the Amazon Resource Name (ARN) of the role, once you've created it. Refer [this document](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html#create-service-account-iam-role) for details on creating an IAM role for Service accounts with eksctl, AWS CLI or AWS Management Console.
   2. Annotate the Kubernetes service account with the ARN of the role that you created with the following command (replace IAM Role ARN).

    ```bash
    kubectl annotate serviceaccount -n default eks-dns-ts \
    eks.amazonaws.com/role-arn=arn:aws:iam::111122223333:role/eks-dns-troubleshooter
    ```

6. Deploy the EKS DNS Troubleshooter with the following command.
```bash
kubectl apply -f https://raw.githubusercontent.com/joshisumit/eks-dns-troubleshooter/v1.1.0/deploy/eks-dns-troubleshooter.yaml
```

7. Verify that pod is deployed

```bash
kubectl logs -l=app=eks-dns-troubleshooter
```

Should display output similar to the following.

```json
time="2020-06-27T15:22:34Z" level=info msg="\n\t-----------------------------------------------------------------------\n\tEKS DNS Troubleshooter\n\tRelease:    v1.1.0\n\tBuild:      git-89606ea\n\tRepository: https://github.com/joshisumit/eks-dns-troubleshooter\n\t-----------------------------------------------------------------------\n\t"
{"file":"main.go:81","func":"main","level":"info","msg":"Running on Kubernetes v1.15.11-eks-af3caf","time":"2020-06-27T15:22:34Z"}
{"file":"main.go:100","func":"main","level":"info","msg":"kube-dns service ClusterIP: 10.100.0.10","time":"2020-06-27T15:22:34Z"}
{"file":"verify-configs.go:42","func":"checkServieEndpoint","level":"info","msg":"Endpoints addresses: {[{192.168.27.238  0xc0004488a0 \u0026ObjectReference{Kind:Pod,Namespace:kube-system,Name:coredns-6658f9f447-gm5sn,UID:75d7a6cf-4092-4501-a56f-d3c859cdc5d0,APIVersion:,ResourceVersion:36301646,FieldPath:,}} {192.168.5.208  0xc0004488b0 \u0026ObjectReference{Kind:Pod,Namespace:kube-system,Name:coredns-6658f9f447-gqqk2,UID:f1864697-db9d-460c-b219-d5fbbe7484b8,APIVersion:,ResourceVersion:36301561,FieldPath:,}}] [] [{dns 53 UDP} {dns-tcp 53 TCP}]}","time":"2020-06-27T15:22:34Z"}
...
{"file":"main.go:173","func":"main","level":"info","msg":"DNS Diagnosis completed. Please check diagnosis report in /var/log/eks-dns-diag-summary.json file.","time":"2020-06-27T15:23:34Z"}
```
Once pod is running it will validate and troubleshoot DNS, which will take around 2 mins and then it will generate the diagnosis report in the JSON format (last line in the above log excerpt).

8. Exec into pod and fetch the diagnosis report

```bash
POD_NAME=$(kubectl get pods -l=app=eks-dns-troubleshooter -o jsonpath='{.items..metadata.name}')
kubectl exec -ti $POD_NAME -- cat /var/log/eks-dns-diag-summary.json | jq
```
OR download the diagnosis report JSON file locally

```bash
kubectl exec -ti $POD_NAME -- cat /var/log/eks-dns-diag-summary.json > eks-dns-diag-summary.json
```


#### _See diagnosis report in JSON format:_


```json
{
  "diagnosisCompletion": true,
  "diagnosisToolInfo": {
    "release": "v1.1.0",
    "repo": "https://github.com/joshisumit/eks-dns-troubleshooter",
    "commit": "git-89606ea"
  },
  "Analysis": {
    "dnstest": "DNS resolution is working correctly in the cluster",
    "naclRules": "naclRules are configured correctly...NOT blocking any DNS communication",
    "securityGroupConfigurations": "securityGroups are configured correctly...not blocking any DNS communication"
  },
  "eksVersion": "v1.16.8-eks-e16311",
  "corednsChecks": {
    "clusterIP": "10.100.0.10",
    "endpointsIP": [
      "192.168.10.9",
      "192.168.17.192"
    ],
    "notReadyEndpoints": [],
    "namespace": "kube-system",
    "imageVersion": "v1.6.6",
    "recommendedVersion": "v1.6.6",
    "dnstestResults": {
      "dnsResolution": "success",
      "description": "tests the internal and external DNS queries against ClusterIP and two Coredns Pod IPs",
      "domainsTested": [
        "amazon.com",
        "kubernetes.default.svc.cluster.local"
      ],
      "detailedResultForEachDomain": [
        {
          "domain": "amazon.com",
          "server": "10.100.0.10:53",
          "result": "success"
        },
        {
          "domain": "amazon.com",
          "server": "192.168.10.9:53",
          "result": "success"
        },
        {
          "domain": "amazon.com",
          "server": "192.168.17.192:53",
          "result": "success"
        },
        {
          "domain": "kubernetes.default.svc.cluster.local",
          "server": "10.100.0.10:53",
          "result": "success"
        },
        {
          "domain": "kubernetes.default.svc.cluster.local",
          "server": "192.168.10.9:53",
          "result": "success"
        },
        {
          "domain": "kubernetes.default.svc.cluster.local",
          "server": "192.168.17.192:53",
          "result": "success"
        }
      ]
    },
    "replicas": 2,
    "podNames": [
      "coredns-76f4cb57b4-25x8d",
      "coredns-76f4cb57b4-2vs9w"
    ],
    "corefile": ".:53 {\n    log\n    errors\n    health\n    kubernetes cluster.local in-addr.arpa ip6.arpa {\n      pods insecure\n      upstream\n      fallthrough in-addr.arpa ip6.arpa\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
    "resolvconf": {
      "SearchPath": [
        "default.svc.cluster.local",
        "svc.cluster.local",
        "cluster.local",
        "eu-west-2.compute.internal"
      ],
      "Nameserver": [
        "10.100.0.10"
      ],
      "Options": [
        "ndots:5"
      ],
      "Ndots": 5
    },
    "errorCheckInCorednsLogs": {
      "errorsInLogs": false
    }
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
    "region": "eu-west-2",
    "securityGroupIds": [
      "eksctl-dev-cluster-nodegroup-ng-1-SG-40FRGTVAFSPT",
      "eksctl-dev-cluster-cluster-ClusterSharedNodeSecurityGroup-1PISH2DINONDS"
    ],
    "clusterName": "dev-cluster",
    "clusterSecurityGroup": "sg-0529eb51ffbae7373"
  }
}
```


## Notes
- Tool tested for EKS version 1.14 onwards
- In order to check errors in coredns pod logs, make sure to enable `log` plugin in the coredns ConfigMap before running the tool.
- It is recommended to use `IAM roles for Service Accounts (IRSA)` to associate the Service Account that the EKS DNS Troubleshooter Deployment runs as with an IAM role that is able to perform these functions. If you are unable to use `IRSA`, you may associate an IAM Policy with the EC2 instance on which the EKS DNS Troubleshooter pod runs.
- Two files are generated inside a pod:
  1.  `/var/log/eks-dns-tool.log` - Tool execution logs which can be used for debugging purpose
  2.  `/var/log/eks-dns-diag-summary.json` - Final Diagnosis result in JSON format
- Once diagnosis is complete, pod will continue to run.
- To rerun the troubleshooting after a diagnosis, exec into running pod and rerun the tool again. Something like:
    `kubectl exec -ti $POD_NAME -- /app/eks-dnshooter`
- Docker image includes common network troubleshooting utility like `curl`, `dig`, `nslookup` etc.

## Contribute
- Any tips/PR on how to make the code cleaner or more idiomatic would be welcome.
- More scenarios for DNS troubleshooting would be welcome.


