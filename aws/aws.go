package aws

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"

	"github.com/sas-fe/cloud-provider-tools/common"
)

// NewClient creates a new EC2 client for server operations
func NewClient() *ec2.EC2 {
	sess, err := session.NewSession()

	if err != nil {
		log.Println("Could not create EC2 client", err)
		return nil
	}

	return ec2.New(sess)
}

// NewRouter creates a new Route53 client for DNS operations
func NewRouter() *route53.Route53 {
	sess, err := session.NewSession()

	if err != nil {
		log.Println("Could not create Route53 client", err)
		return nil
	}

	return route53.New(sess)
}

// Provider implements common.CloudProvider
type Provider struct {
	client *ec2.EC2         // EC2 client
	router *route53.Route53 // Route53 client
	domain string           // server domain name
	alloc  string           // IP allocation ID, needed for later deletion
	zone   string           // hosted zone ID, needed for DNS operations
}

// NewProvider returns a new Provider instance
func NewProvider(domain string) *Provider {
	return &Provider{NewClient(), NewRouter(), domain, "", ""}
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

	log.Println("Creating server instance")
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

	log.Println("Created server instance", *runResult.Instances[0].InstanceId)

	instanceID = *runResult.Instances[0].InstanceId

	time.Sleep(60 * time.Second)

	allocRes, _, _ := p.CreateIPAddress(ctx, instanceID)
	p.alloc = *allocRes.AllocationId
	instanceIP = *allocRes.PublicIp

	return &common.CreateServerResponse{
		Name:     name,
		ServerID: instanceID,
		ServerIP: instanceIP,
	}, nil
}

// CreateIPAddress allocates and associates an Elastic IP to a server instance
func (p *Provider) CreateIPAddress(ctx context.Context, instanceID string) (*ec2.AllocateAddressOutput, *ec2.AssociateAddressOutput, error) {
	svc := p.client

	log.Println("Allocating IP address")
	allocRes, err := svc.AllocateAddress(&ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	})
	if err != nil {
		log.Println("Unable to allocate IP address,", err)
	}

	log.Println("Associating IP address to instance", instanceID)
	assocRes, err := svc.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: allocRes.AllocationId,
		InstanceId:   aws.String(instanceID),
	})
	if err != nil {
		log.Println("Unable to associate IP address with", instanceID, err)
	}

	log.Printf("Successfully allocated %s with instance %s.\n\tallocation id: %s, association id: %s\n", *allocRes.PublicIp, instanceID, *allocRes.AllocationId, *assocRes.AssociationId)
	return allocRes, assocRes, nil
}

// CreateDNSRecord creates a DNS A Record on AWS
func (p *Provider) CreateDNSRecord(ctx context.Context, subDomain string, IP string) (*common.CreateDNSRecordResponse, error) {
	svc := p.router

	log.Println("Creating DNS A record for", subDomain)
	request := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(subDomain + "." + p.domain),
						Type: aws.String("A"),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(IP),
							},
						},
						TTL: aws.Int64(300),
					},
				},
			},
		},
		HostedZoneId: aws.String(p.zone),
	}
	resp, err := svc.ChangeResourceRecordSets(request)
	if err != nil {
		log.Println("Unable to create DNS Record", err)
	}

	return &common.CreateDNSRecordResponse{
		SubDomain:   subDomain,
		SubDomainID: *resp.ChangeInfo.Id,
		SubDomainIP: IP,
	}, nil
}

// CreateHostedZone creates a Route53 HostedZone
func (p *Provider) CreateHostedZone(ctx context.Context, server *common.CreateServerResponse) error {
	log.Println("Creating hosted zone", p.domain)

	params := &route53.CreateHostedZoneInput{
		CallerReference: aws.String(server.ServerID.(string)),
		Name:            aws.String(p.domain),
	}
	resp, err := p.router.CreateHostedZone(params)
	if err != nil {
		log.Println("Unable to created hosted zone,", err)
	}
	p.zone = *resp.HostedZone.Id

	return nil
}

// RemoveServer removes an EC2 instance on AWS
func (p *Provider) RemoveServer(ctx context.Context, server *common.CreateServerResponse) error {
	svc := p.client

	log.Println("Removing server", server.ServerID)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(server.ServerID.(string)),
		},
	}
	_, err := svc.TerminateInstances(input)
	if err != nil {
		log.Println("Unable to remove server", err)
	}

	err = p.RemoveIPAddress(ctx)
	if err != nil {
		log.Println("Unable to release IP address", err)
	}

	log.Println("Done")

	return nil
}

// RemoveIPAddress dissociates and releases an Elastic IP
func (p *Provider) RemoveIPAddress(ctx context.Context) error {
	svc := p.client

	_, err := svc.ReleaseAddress(&ec2.ReleaseAddressInput{
		AllocationId: aws.String(p.alloc),
	})
	if err != nil {
		log.Println("Unable to release IP address,", err)
	}

	return nil
}

// RemoveDNSRecord removes a DNS A Record from AWS
func (p *Provider) RemoveDNSRecord(ctx context.Context, subDomain *common.CreateDNSRecordResponse) error {
	svc := p.router

	log.Println("Removing DNS A record for", subDomain.SubDomain)
	request := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("DELETE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(subDomain.SubDomain + "." + p.domain),
						Type: aws.String("A"),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(subDomain.SubDomainIP),
							},
						},
						TTL: aws.Int64(300),
					},
				},
			},
		},
		HostedZoneId: aws.String(p.zone),
	}
	_, err := svc.ChangeResourceRecordSets(request)
	if err != nil {
		log.Println("Unable to delete DNS Record", err)
	}

	return nil
}

// RemoveHostedZone removes an empty Route53 HostedZone
func (p *Provider) RemoveHostedZone(ctx context.Context) error {
	svc := p.router

	log.Println("Removing hosted zone", p.domain)
	params := &route53.DeleteHostedZoneInput{
		Id: aws.String(p.zone),
	}
	_, err := svc.DeleteHostedZone(params)

	if err != nil {
		panic(err)
	}

	return nil
}

// CreateServerGroup unimplemented for AWS
func (p *Provider) CreateServerGroup(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerGroupResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveServerGroup unimplemented for AWS
func (p *Provider) RemoveServerGroup(ctx context.Context, group *common.CreateServerGroupResponse) error {
	return errors.New("Unimplemented")
}

// CreateK8s unimplemented for AWS
func (p *Provider) CreateK8s(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateK8sResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveK8s unimplemented for AWS
func (p *Provider) RemoveK8s(ctx context.Context, k8s *common.CreateK8sResponse) error {
	return errors.New("Unimplemented")
}

// CreateStaticIP unimplemented for AWS
func (p *Provider) CreateStaticIP(ctx context.Context, name string) (*common.CreateStaticIPResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveStaticIP unimplemented for AWS
func (p *Provider) RemoveStaticIP(ctx context.Context, staticIP *common.CreateStaticIPResponse) error {
	return errors.New("Unimplemented")
}
