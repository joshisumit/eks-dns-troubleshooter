package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"

	"errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
)

const (
	maxRetries       = 10
	resourceID       = "resource-id"
	resourceKey      = "key"
	tagKeyMatchValue = "kubernetes.io/cluster/"
)

type ec2MetdataClient struct {
	//instanceIdentityDocument ec2metadata.EC2InstanceIdentityDocument
	client ec2metadata.EC2Metadata
}

type ec2Client struct {
	//client ec2.EC2
	ec2ServiceClient ec2iface.EC2API
}

// clusterSGRulesCheck stores details of Cluster SG ID verification
// Example: {isClusterSGRuleCorrect: true, inboundRule: {"isValid": "true", "details": "dfsdffd"}, outboundRule: {"isValid": "true", "details": "dfsdffd"}}
type clusterSGRulesCheck struct {
	IsClusterSGRuleCorrect bool
	InboundRule            map[string]string
	OutboundRule           map[string]string
}

//ClusterInfo stores details of EKS cluster
type ClusterInfo struct {
	SgRulesCheck             clusterSGRulesCheck                     `json:"securityGroupChecks"`
	NaclRulesCheck           bool                                    `json:"naclRulesCheck"`
	Region                   string                                  `json:"region"`
	SecurityGroupIds         []string                                `json:"securityGroupIds"`
	ClusterName              string                                  `json:"clusterName"`
	ClusterSGID              string                                  `json:"clusterSecurityGroup"`
	TagList                  []map[string]string                     `json:"tagList,omitempty"`
	InstanceIdentityDocument ec2metadata.EC2InstanceIdentityDocument `json:"instanceIdentityDocument"`
	ClusterDetails           *eks.Cluster                            `json:"clusterDetails"`
}

func getInstanceIdentityDocument() (*ClusterInfo, error) {
	//create session
	sess := session.Must(session.NewSession())
	log.Printf("Session: %v\n", *sess)

	//create  metadata client
	metadataClient := ec2metadata.New(sess)

	doc, err := metadataClient.GetInstanceIdentityDocument()
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return &ClusterInfo{}, err
	}
	log.Printf("EC2 Instacne ID doc: %+v\n\n", doc)
	log.Printf("Region from EC2 Instacne ID doc: %+v\n", doc.Region)

	return &ClusterInfo{
		InstanceIdentityDocument: doc,
	}, nil

}

func newEC2Client(region string) (*ec2Client, error) {
	ec2session := session.Must(session.NewSession())
	ec2cl := ec2.New(ec2session, aws.NewConfig().WithMaxRetries(maxRetries).WithRegion(region))

	return &ec2Client{
		ec2ServiceClient: ec2cl,
	}, nil
}

//getClusterName fetches EKS cluster name from the EC2 worker node tags
func (e *ec2Client) getClusterName(resourceId string) (string, error) {
	var clusterName string

	input := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(resourceID),
				Values: []*string{
					aws.String(resourceId),
				},
			},
		},
	}

	//Describe Tags of a EC2 worker node
	tagsList, err := e.ec2ServiceClient.DescribeTags(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return "", err
	}
	if len(tagsList.Tags) < 1 {
		return "", fmt.Errorf("There are no tags attached to EC2 instance")
	}

	//parse all the tags & fetch clusterName from the tag "kubernetes.io/cluster/"
	for _, tagDetails := range tagsList.Tags {
		if strings.Contains(aws.StringValue(tagDetails.Key), tagKeyMatchValue) {
			clusterName = strings.Split(aws.StringValue(tagDetails.Key), "/")[2]
		}
	}
	if clusterName == "" {
		return "", fmt.Errorf("Error finding clustername...Reuired tag not found")
	}

	return clusterName, nil
}

func (w *ClusterInfo) getAttachedSG() ([]string, error) {
	//secGroupIds := make([]string, 0, 4)

	metadataSession := session.Must(session.NewSession())
	metadataClient := ec2metadata.New(metadataSession)

	//check if metadata is available and to to detect if pod is running on EC2
	isMetadataAvailable := metadataClient.Available()
	if !isMetadataAvailable {
		return nil, errors.New("Metdata is not available")
	}

	sgList, err := metadataClient.GetMetadata("security-groups")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}
	//fmt.Printf("SG attached to EC2 instance: %v %T\n", sgList, sgList)
	secGroupIds := strings.Fields(sgList)
	return secGroupIds, nil
}

//getClusterDetails executes describeCluster API call and returns cluster details and ClusterSGID
func (w *ClusterInfo) getClusterDetails(clusterName string, region string) (*eks.Cluster, string, error) {
	eksSession := session.Must(session.NewSession())
	eksClient := eks.New(eksSession, aws.NewConfig().WithMaxRetries(maxRetries).WithRegion(region))

	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	result, err := eksClient.DescribeCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return &eks.Cluster{}, "", err
	}

	log.Infof("Cluster Details: %v", result)
	clusterDe := result.Cluster
	//log.Infof("clusterDe: %+v \n\n%T \n\nclusterSGID:%v", clusterDe, clusterDe, aws.StringValue(clusterDe.ResourcesVpcConfig.ClusterSecurityGroupId))

	return clusterDe, aws.StringValue(clusterDe.ResourcesVpcConfig.ClusterSecurityGroupId), nil

}

// getSecurityGrupRules returns SG rules based on sg-id or sg-name
func getSecurityGrupRules(sgFilter string, region string) (*ec2.SecurityGroup, error) {
	ec2Client, err := newEC2Client(region)
	fmt.Printf("EC2 Client: %v %T\n\n", ec2Client, ec2Client)

	//input := &ec2.DescribeSecurityGroupsInput{}
	var input *ec2.DescribeSecurityGroupsInput

	//if sgFilter starts with sg- pass sgID in DescribeSecurityGroups()
	if strings.HasPrefix(sgFilter, "sg-") {
		input = &ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{
				aws.String(sgFilter),
			},
		}
	} else {
		//if sgFilter does not starts with sg- pass sgName in DescribeSecurityGroups()
		input = &ec2.DescribeSecurityGroupsInput{
			GroupNames: []*string{
				aws.String(sgFilter),
			},
		}
	}

	result, err := ec2Client.ec2ServiceClient.DescribeSecurityGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Infof(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Infof(err.Error())
		}
		return nil, err
	}

	sgs := result.SecurityGroups
	//log.Infof("sgList: %v %T", sgs, sgs)

	//log.Infof("overall SG:%v \n\n Inbound rules: %v\n\n", sgs[0], sgs[0].IpPermissions)
	return sgs[0], nil
}

//verifyClusterSGRules verifies that clusterSG is configured correctly
//returns true, true, nil if inbound and outbound rules are configured correctly
//to make sure that inter-worker node communication is allowed...clusterSecurityGroup should be attached to worker node and clusterSG should allow all the traffic from self
func verifyClusterSGRules(clusterSGID string, region string) (bool, bool, error) {
	var isInBoundRulesCorrect, isOutBoundRulesCorrect bool

	//1. Fetch rules of Cluster SG
	sgrules, err := getSecurityGrupRules(clusterSGID, region)
	if err != nil {
		log.Printf("Unable to evaluate the rules of Cluster SG %v\n", err)
		return false, false, err
	}
	log.Infof("sgrules: %v\n", sgrules)

	//2. evaluate inbound rules - checks if rule self-references cluster SG ID
	for _, rules := range sgrules.IpPermissions {
		for _, rule := range rules.UserIdGroupPairs {
			if *rule.GroupId == clusterSGID && *rules.IpProtocol == "-1" {
				isInBoundRulesCorrect = true
			}
		}
	}

	//3. evaluate outbound rules
	for _, rules := range sgrules.IpPermissionsEgress {
		for _, rule := range rules.IpRanges {
			if *rule.CidrIp == "0.0.0.0/0" && *rules.IpProtocol == "-1" {
				isOutBoundRulesCorrect = true
			}
		}
	}
	log.Infof("%v %v\n\n", isInBoundRulesCorrect, isOutBoundRulesCorrect)

	return isInBoundRulesCorrect, isOutBoundRulesCorrect, nil
}

//verifyNaclRules checks NACL rules for the VPC associated with the EKS cluster
//checks if NACL allows outbound TCP and UDP access on port 53 for DNS resolution
//returns true if all good
// requires IAM policy
func verifyNaclRules(region string, vpcid string) (bool, error) {

	ec2Client, err := newEC2Client(region)

	//    NetworkAclIds: []*string{
	//aws.String("acl-5fb85d36"),
	//},
	input := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcid),
				},
			},
		},
	}

	result, err := ec2Client.ec2ServiceClient.DescribeNetworkAcls(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Infof(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Infof(err.Error())
		}
		return false, err
	}

	nacls := result.NetworkAcls
	log.Infof("NACL rules list: %v\n", nacls)

	var isPort53EgressAllowed bool

	//TODO: remove nested-if statements
	if len(nacls) > 0 {
		for _, nacl := range nacls {
			naclID, entries := nacl.NetworkAclId, nacl.Entries
			log.Infof("Evaluating NACL: %v\n\n", aws.StringValue(naclID))
			for _, rule := range entries {
				//ony check egress rules if they allow outbound access on port 53
				if aws.BoolValue(rule.Egress) {
					if aws.StringValue(rule.RuleAction) == "allow" && rule.PortRange == nil {
						log.Infof("Rule number: %v is not blocking any egress traffic", aws.Int64Value(rule.RuleNumber))
						isPort53EgressAllowed = true
						//continue
					} else if aws.Int64Value(rule.RuleNumber) == 32767 {
						//break
						log.Infof("Hit the default egress rule of NACL...continuing with next rule")
						continue
					} else if aws.StringValue(rule.RuleAction) == "allow" {
						log.Infof("Evaluating rule number: %v\n\n", aws.Int64Value(rule.RuleNumber))
						from, to := aws.Int64Value(rule.PortRange.From), aws.Int64Value(rule.PortRange.To)
						//checks if from...to range contains 53
						for _, port := range makeRange(from, to) {
							//check if UDP Protocol (number 17) port 53 is allowed
							if port == 53 && aws.StringValue(rule.Protocol) == "17" {
								log.Infof("UDP port 53 is allowed in the NACL rule number: %v", aws.Int64Value(rule.RuleNumber))
								isPort53EgressAllowed = true
								continue
							}
						}
					}
				}
			}
		}
	}
	if !isPort53EgressAllowed {
		log.Infof("NACL rules are not allowing egress for port 53")
	}
	//naclID, entries := nacls[0].NetworkAclId, nacls[0].Entries

	return isPort53EgressAllowed, nil
}

//makeRange returns slice of integers between specific range
func makeRange(min, max int64) []int64 {
	a := make([]int64, max-min+1)
	for i := range a {
		a[i] = min + int64(i)
	}
	return a
}

//DiscoverClusterInfo checks EKS cluster resources
//like ClusterSecurityGroup, NACL
func DiscoverClusterInfo() (*ClusterInfo, error) {

	wkr1, err := getInstanceIdentityDocument()
	if err != nil {
		log.Errorf("Failed to fetch instanceID document")
		return nil, err
	}
	log.Infof("Worker node Info: %v", wkr1)
	log.Infof("Worker node Info: %+v", wkr1.InstanceIdentityDocument)

	region := wkr1.InstanceIdentityDocument.Region
	log.Infof("Region is: %v", region)

	// err = getTags(region, wkr.instanceIdentityDocument.InstanceID)
	// if err != nil {
	// 	log.Errorf("Failed to fetch tags of an instance : ", err)
	// }

	ec2Client, _ := newEC2Client(region)
	fmt.Printf("EC2 Client: %v %T\n\n", ec2Client, ec2Client)

	wkr := ClusterInfo{}
	log.Infof("Fetching Clustername...")
	clusterName, err := ec2Client.getClusterName(wkr1.InstanceIdentityDocument.InstanceID)
	if err != nil {
		log.Errorf(`Error finding required tag "kubernetes.io/cluster/clusterName" on EC2 instance : `, err)
		return nil, err
	}
	wkr.ClusterName = clusterName
	log.Infof("Clustername is :%v", clusterName)

	log.Infof("Fetching SGs attached to an EC2 instance")
	sgID, err := wkr.getAttachedSG()
	if err != nil {
		log.Printf("Unable to retrieve the SGs attached to the EC2 instance %v\n", err)
		return nil, err
	}
	log.Infof("SGs attached to instance: %v", sgID)
	wkr.SecurityGroupIds = sgID

	//Get EKS cluster details
	log.Infof("Fetching details of EKS cluster %q using DescribeCluster API", clusterName)
	wkr.ClusterDetails, wkr.ClusterSGID, err = wkr.getClusterDetails(clusterName, region)
	if err != nil {
		log.Printf("Unable to retrieve cluster Details %v\n", err)
		return nil, err
	}
	log.Infof("details: %v %T", *wkr.ClusterDetails, wkr.ClusterDetails)

	log.Infof("Evaluating Cluster Security-Group ID")
	inbound, outbound, err := verifyClusterSGRules(wkr.ClusterSGID, region)
	if err != nil {
		log.Printf("Unable to evaluate the rules of Cluster SG %v\n", err)
		return nil, err
	}

	//wkr.isClusterSGRulesCorrect = make(map[bool]string)
	wkr.SgRulesCheck.InboundRule = make(map[string]string)
	wkr.SgRulesCheck.OutboundRule = make(map[string]string)
	if !inbound {
		wkr.SgRulesCheck.IsClusterSGRuleCorrect = false
		wkr.SgRulesCheck.InboundRule["isValid"] = "false"
		wkr.SgRulesCheck.InboundRule["details"] = fmt.Sprintf(`cluster Security Group %q is not configured correctly, 
		please make sure that cluster Security Gorup has inbound rule which references itself...
		Refer: https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html#cluster-sg`, wkr.ClusterSGID)
		log.Infof("%v", wkr.SgRulesCheck)
	} else if !outbound {
		wkr.SgRulesCheck.IsClusterSGRuleCorrect = false
		wkr.SgRulesCheck.OutboundRule["isValid"] = "false"
		wkr.SgRulesCheck.OutboundRule["details"] = fmt.Sprintf(`cluster Security Group %q is not configured correctly, 
		outbound rules are not allowing all traffic...For more details: https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html#cluster-sg`, wkr.ClusterSGID)
		log.Infof("%v", wkr.SgRulesCheck)
	} else {
		wkr.SgRulesCheck.IsClusterSGRuleCorrect = true
		wkr.SgRulesCheck.InboundRule["isValid"] = "true"
		wkr.SgRulesCheck.OutboundRule["isValid"] = "true"
		log.Infof("clustreSG %q is configured correctly, it references itself", wkr.ClusterSGID)
		log.Infof("%v", wkr.SgRulesCheck)
	}

	isNaclOk, err := verifyNaclRules(region, *wkr.ClusterDetails.ResourcesVpcConfig.VpcId)
	if err != nil {
		log.Errorf("Unable to retrieve NACL rules %v\n", err)
		return nil, err
	}
	wkr.NaclRulesCheck = isNaclOk
	log.Infof("NACL rules are: %v", isNaclOk)

	log.Infof("worker node struct: %+v", wkr)

	return &wkr, nil

}
