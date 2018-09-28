package aws

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/sas-fe/cloud-provider-tools/common"
)

// type tokenSource struct {
// 	AccessToken string
// }

// func (t *tokenSource) Token() (*oauth2.Token, error) {
// 	token := &oauth2.Token{
// 		AccessToken: t.AccessToken,
// 	}
// 	return token, nil
// }

// NewClient creates a new EC2 client
func NewClient() *ec2.EC2 {
	// Create session
	sess, err := session.NewSession()
	return ec2.New(sess)
}

// Provider implements common.CloudProvider
type Provider struct {
	client *ec2.EC2
}

// NewProvider returns a new Provider instance
func NewProvider() *Provider {
	return &Provider{NewClient()}
}

// CreateServer creates an EC2 instance on AWS
func (p *Provider) CreateServer(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerResponse, error) {
	var instanceID string
	var instanceIP string

	s := &common.ServerInfo{
		Name: name,
	}

	for _, opt := range opts {
		opt.Set(s)
	}

	var imageIDStr string
	if len(s.Image) == 0 {
		imageIDStr = os.Getenv("AWS_IMAGE_ID")
	} else {
		imageIDStr = s.Image
	}

	svc := p.client

	fmt.Println("Creating EC2 instance...")

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String(imageIDStr),
		InstanceType: aws.String(s.Size),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
	})

	if err != nil {
		log.Println("Could not create instance", err)
		return nil, err
	}

	log.Println("Created instance", *runResult.Instances[0].InstanceId)

	instanceID = *runResult.Instances[0].InstanceId

	time.Sleep(60 * time.Second)

	allocRes, err := svc.AllocateAddress(&ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	})
	if err != nil {
		log.Println("Unable to allocate IP address, %v", err)
	}

	assocRes, err := svc.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: allocRes.AllocationId,
		InstanceId:   aws.String(instanceID),
	})
	if err != nil {
		log.Println("Unable to associate IP address with %s, %v",
			instanceID, err)
	}

	fmt.Printf("Successfully allocated %s with instance %s.\n\tallocation id: %s, association id: %s\n", *allocRes.PublicIp, instanceID, *allocRes.AllocationId, *assocRes.AssociationId)

	instanceIP = *allocRes.PublicIp

	return &common.CreateServerResponse{
		Name:     name,
		ServerID: instanceID,
		ServerIP: instanceIP,
	}, nil
}

// CreateDNSRecord creates a DNS A Record on AWS
func (p *Provider) CreateDNSRecord(ctx context.Context, subDomain string, IP string) (*common.CreateDNSRecordResponse, error) {
	// domainRequest := &godo.DomainRecordEditRequest{
	// 	Type: "A",
	// 	Name: subDomain,
	// 	Data: IP,
	// }
	// domainRecord, _, err := p.client.Domains.CreateRecord(ctx, p.domain, domainRequest)
	// if err != nil {
	// 	return nil, err
	// }

	// return &common.CreateDNSRecordResponse{
	// 	SubDomain:   subDomain,
	// 	SubDomainID: domainRecord.ID,
	// }, nil
	return nil, nil
}

// RemoveServer removes an EC2 instance on AWS
func (p *Provider) RemoveServer(ctx context.Context, serverID interface{}) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(serverID.(string)),
		},
	}
	svc := p.client
	result, err := svc.StopInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok {
		result, err = svc.StopInstances(input)
		if err != nil {
			fmt.Println("Error", err)
		} else {
			fmt.Println("Success", result.StoppingInstances)
		}
	} else {
		fmt.Println("Error", awsErr)
	}

	fmt.Println("Done")

	return nil
}

// RemoveDNSRecord removes a DNS A Record from AWS
func (p *Provider) RemoveDNSRecord(ctx context.Context, subDomainID interface{}) error {
	// intSubDomainID, ok := subDomainID.(int)
	// if !ok {
	// 	return fmt.Errorf("%v is not an int", subDomainID)
	// }

	// fmt.Println("Deleting Domain Record...")
	// _, err := p.client.Domains.DeleteRecord(ctx, p.domain, intSubDomainID)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("Done")

	return nil
}
