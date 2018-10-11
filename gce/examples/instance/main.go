package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/common"
	"github.com/sas-fe/cloud-provider-tools/gce"
)

func main() {
	adc := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if len(adc) == 0 {
		panic("$GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	projectID := os.Getenv("GCP_PROJECT")
	if len(projectID) == 0 {
		panic("$GCP_PROJECT not set")
	}

	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}

	dnsZone := os.Getenv("GCP_DNS_ZONE")
	if len(domain) == 0 {
		panic("$GCP_DNS_ZONE not set")
	}

	ctx := context.TODO()

	p, err := gce.NewProvider(projectID, domain, dnsZone)
	if err != nil {
		panic(err)
	}

	startupScript := `#!/bin/bash	
	curl -fsSL https://get.docker.com -o get-docker.sh
	sudo sh get-docker.sh`

	fmt.Println("Creating Instance")
	serverResp, err := p.CreateServer(
		ctx,
		"test-ondemand",
		common.ServerRegion("us-east1-c"),
		common.ServerSize("n1-highcpu-4"),
		common.ServerUserData(startupScript),
		common.ServerTags([]string{"OnDemand"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(serverResp)

	subDomain := serverResp.Name + "-" + serverResp.ServerID.(string) + "." + "instances"

	fmt.Println("Creating DNS record")
	dnsResp, err := p.CreateDNSRecord(ctx, subDomain, serverResp.ServerIP)
	if err != nil {
		panic(err)
	}

	fmt.Println(dnsResp)

	fmt.Println("Sleeping for 300 seconds")
	time.Sleep(300 * time.Second)

	fmt.Println("Removing Instance")
	err2 := p.RemoveServer(ctx, serverResp)
	if err2 != nil {
		panic(err2)
	}

	fmt.Println("Removing DNS record")
	err3 := p.RemoveDNSRecord(ctx, dnsResp)
	if err3 != nil {
		panic(err3)
	}
}
