# cloud-provider-tools
[![GoDoc](https://godoc.org/github.com/sas-fe/cloud-provider-tools?status.svg)](https://godoc.org/github.com/sas-fe/cloud-provider-tools)

## Usage

```go
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

	clusterName := "test-k8s"
	subDomain := clusterName + "." + "instances"

	fmt.Println("Acquiring global static IP")
	ipResp, err := p.CreateStaticIP(ctx, clusterName, &common.StaticIPRequest{IPType: common.GLOBAL, Region: "us-east1"})
	if err != nil {
		panic(err)
	}
	fmt.Println(ipResp)

	fmt.Println("Creating DNS record")
	dnsResp, err := p.CreateDNSRecord(ctx, subDomain, ipResp.StaticIP)
	if err != nil {
		panic(err)
	}
	fmt.Println(dnsResp)

	fmt.Println("Creating Cluster")
	k8sResp, err := p.CreateK8s(
		ctx,
		clusterName,
		common.ServerRegion("us-east1-c"),
		common.ServerSize("n1-standard-1"),
		common.ServerTags([]string{"OnDemand"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(k8sResp)
	fmt.Println(k8sResp.Credentials)

	fmt.Println("Sleeping for 300 seconds")
	time.Sleep(300 * time.Second)

	fmt.Println("Removing Cluster")
	err2 := p.RemoveK8s(ctx, k8sResp)
	if err2 != nil {
		panic(err2)
	}

	fmt.Println("Removing DNS record")
	err3 := p.RemoveDNSRecord(ctx, dnsResp)
	if err3 != nil {
		panic(err3)
	}

	fmt.Println("Removing static IP")
	err4 := p.RemoveStaticIP(ctx, ipResp)
	if err4 != nil {
		panic(err4)
	}
}
```

## DigitalOcean Provider Settings

### Creating DigitalOcean Provider Instance
If using `cpt.NewCloudProvider(cpt.DIGITALOCEAN)`, `$DOMAIN` and `$DO_TOKEN` should be set
to the target domain and digitalocean API token respectively. Those can also be manually
passed in via `digitalocean.NewProvider()`.

### Creating DigitalOcean Droplets
The digitalocean provider recognizes several `common.ServerOption`s to customize droplets.
Supported options include:
- Region: e.g. `common.ServerRegion("nyc1")`
- Size: e.g. `common.ServerSize("s-1vcpu-1gb")`
- Image ID: e.g. `common.ServerImage("31734516")`
- Startup Script/User Data: e.g. `common.ServerScript("#!/bin/bash\necho 'Hello, World!'")`
- Tags: e.g. `common.ServerTags([]string{"OnDemand"})`


## Google Compute Engine Provider Settings

### Creating GCE Provider Instance
`cpt.NewCloudProvider(cpt.GCE)` requires several environment variables to be set:
- `$GOOGLE_APPLICATION_CREDENTIALS`: service account file path for google ADC
- `$GCP_PROJECT`: GCP project to use
- `$GCP_DNS_ZONE`: Cloud DNS manage zone to use 
- `$DOMAIN`: base domain name
Those can also be manually passed in via `gce.NewProvider()`.

### Creating GCE Instances
The GCE provider recognizes several `common.ServerOption`s to customize instances.
Supported options include:
- Zone: e.g. `common.ServerRegion(us-east1-c)`
- Machine Type: e.g. `common.ServerSize("n1-highcpu-4")`
- Source Image: e.g. `common.ServerImage("projects/ubuntu-os-cloud/global/images/ubuntu-1604-xenial-v20180912")`
- Startup Script/User Data: e.g. `common.ServerScript("#!/bin/bash\necho 'Hello, World!'")`
- Tags: e.g. `common.ServerTags([]string{"OnDemand"})`
