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

type workerNodeInfo struct {
	instanceIdentityDocument ec2metadata.EC2InstanceIdentityDocument
	securityGroupIds         []string
	clusterName              string
}

func getInstanceIdentityDocument() (*workerNodeInfo, error) {
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
		return &workerNodeInfo{}, err
	}
	log.Printf("EC2 Instacne ID doc: %+v\n\n", doc)
	log.Printf("Region from EC2 Instacne ID doc: %+v\n", doc.Region)

	return &workerNodeInfo{
		instanceIdentityDocument: doc,
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

func (w *workerNodeInfo) getAttachedSG() ([]string, error) {
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

//DiscoverClusterInfo hkhkh
func DiscoverClusterInfo() {
	// wkr, err := getInstanceIdentityDocument()
	// if err != nil {
	// 	log.Errorf("Failed to fetch instanceID document")
	// }
	// log.Infof("Worker node Info: %v", wkr)
	// log.Infof("Worker node Info: %+v", wkr.instanceIdentityDocument)

	// region := wkr.instanceIdentityDocument.Region
	// log.Infof("Region is: %v", region)

	// ec2Client, err := newEC2Client(region)
	// fmt.Printf("EC2 Client: %v %T\n\n", ec2Client, ec2Client)

	// log.Infof("Fetching Clustername...")
	// clusterName, err := ec2Client.getClusterName(wkr.instanceIdentityDocument.InstanceID)
	// if err != nil {
	// 	log.Errorf(`Error finding required tag "kubernetes.io/cluster/clusterName" on EC2 instance : `, err)
	// 	//return err
	// }
	// log.Infof("Clustername is :%v", clusterName)

	// log.Infof("Fetching SGs attached to an EC2 instance")
	// sgID, err := getAttachedSG()
	// if err != nil {
	// 	fmt.Printf("Unable to retrieve the SGs attached to the EC2 instance %v\n", err)
	// }
	// log.Infof("SGs attached to instance: %v", sgID)
	//=============================
	wkr1, err := getInstanceIdentityDocument()
	if err != nil {
		log.Errorf("Failed to fetch instanceID document")
	}
	log.Infof("Worker node Info: %v", wkr1)
	log.Infof("Worker node Info: %+v", wkr1.instanceIdentityDocument)

	region := wkr1.instanceIdentityDocument.Region
	log.Infof("Region is: %v", region)

	// err = getTags(region, wkr.instanceIdentityDocument.InstanceID)
	// if err != nil {
	// 	log.Errorf("Failed to fetch tags of an instance : ", err)
	// }

	ec2Client, err := newEC2Client(region)
	fmt.Printf("EC2 Client: %v %T\n\n", ec2Client, ec2Client)

	wkr := workerNodeInfo{}
	log.Infof("Fetching Clustername...")
	clusterName, err := ec2Client.getClusterName(wkr1.instanceIdentityDocument.InstanceID)
	if err != nil {
		log.Errorf(`Error finding required tag "kubernetes.io/cluster/clusterName" on EC2 instance : `, err)
		//return err
	}
	wkr.clusterName = clusterName
	log.Infof("Clustername is :%v", clusterName)

	log.Infof("Fetching SGs attached to an EC2 instance")
	sgID, err := wkr.getAttachedSG()
	if err != nil {
		log.Printf("Unable to retrieve the SGs attached to the EC2 instance %v\n", err)
		//return err
	}
	log.Infof("SGs attached to instance: %v", sgID)
	wkr.securityGroupIds = sgID

	log.Infof("worker node struct: %v", wkr)

}
