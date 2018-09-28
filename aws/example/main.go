package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/aws"
	"github.com/sas-fe/cloud-provider-tools/common"
)

func main() {
	awsToken := os.Getenv("AWS_ACCESS_KEY_ID")
	if len(awsToken) == 0 {
		panic("$AWS_ACCESS_KEY_ID not set")
	}

	awsKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if len(awsKey) == 0 {
		panic("$AWS_SECRET_ACCESS_KEY not set")
	}

	p := aws.NewProvider()

	ctx := context.TODO()

	startupScript := `#!/bin/bash	
	docker run -p 9999:9999 -d --rm sasfe/sas4c:python-only`

	serverResp, err := p.CreateServer(
		ctx,
		"test-ondemand",
		common.ServerRegion("us-east-2"),
		common.ServerSize("t2.micro"),
		common.ServerUserData(startupScript),
		common.ServerTags([]string{"OnDemand"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(serverResp)

	// subDomain := serverResp.Name + "-" + strconv.Itoa(serverResp.ServerID.(int)) + "." + "instances"

	// dnsResp, err := p.CreateDNSRecord(ctx, subDomain, serverResp.ServerIP)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(dnsResp)

	fmt.Println("Sleeping for 120 seconds")
	time.Sleep(120 * time.Second)

	err2 := p.RemoveServer(ctx, serverResp.ServerID)
	if err2 != nil {
		panic(err2)
	}

	// err3 := p.RemoveDNSRecord(ctx, dnsResp.SubDomainID)
	// if err3 != nil {
	// 	panic(err3)
	// }
}
