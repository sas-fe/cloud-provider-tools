package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/aws"
	"github.com/sas-fe/cloud-provider-tools/common"
)

func main() {
	// Key pair provided by AWS
	awsToken := os.Getenv("AWS_ACCESS_KEY_ID")
	if len(awsToken) == 0 {
		panic("$AWS_ACCESS_KEY_ID not set")
	}

	awsKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if len(awsKey) == 0 {
		panic("$AWS_SECRET_ACCESS_KEY not set")
	}

	// AWS operating region (e.g. us-east-2)
	awsRegion := os.Getenv("AWS_REGION")
	if len(awsRegion) == 0 {
		panic("$AWS_REGION not set")
	}

	// Server image ID for creating instances
	awsImageId := os.Getenv("AWS_IMAGE_ID")
	if len(awsImageId) == 0 {
		panic("$AWS_IMAGE_ID not set")
	}

	// Domain to be associated with server instance
	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}

	p := aws.NewProvider(domain)

	ctx := context.TODO()

	startupScript := `#!/bin/bash	
	docker run -p 9999:9999 -d --rm sasfe/sas4c:python-only`

	// Server/IP creation
	serverResp, err := p.CreateServer(
		ctx,
		"test-ondemand",
		common.ServerRegion(awsRegion),
		common.ServerSize("t2.micro"),
		common.ServerUserData(startupScript),
		common.ServerTags([]string{"OnDemand"}),
	)

	if err != nil {
		panic(err)
	}

	// Hosted zone creation
	err2 := p.CreateHostedZone(ctx, serverResp)
	if err2 != nil {
		panic(err2)
	}

	// DNS record creation
	subDomain := serverResp.Name + "-" + serverResp.ServerID.(string) + "." + "instances"
	dnsResp, err3 := p.CreateDNSRecord(ctx, subDomain, serverResp.ServerIP)
	if err3 != nil {
		panic(err3)
	}

	// Intermission
	log.Println("Sleeping for 120 seconds")
	time.Sleep(120 * time.Second)

	// DNS record deletion
	err4 := p.RemoveDNSRecord(ctx, dnsResp)
	if err4 != nil {
		panic(err4)
	}

	// Hosted zone deletion
	err5 := p.RemoveHostedZone(ctx)
	if err5 != nil {
		panic(err5)
	}

	// Server/IP deletion
	err6 := p.RemoveServer(ctx, serverResp)
	if err6 != nil {
		panic(err6)
	}
}
