# cloud-provider-tools
[![GoDoc](https://godoc.org/github.com/sas-fe/cloud-provider-tools?status.svg)](https://godoc.org/github.com/sas-fe/cloud-provider-tools)

## Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/sas-fe/cloud-provider-tools"
    "github.com/sas-fe/cloud-provider-tools/common"
)

func main() {
    p, err := cpt.NewCloudProvider(cpt.DIGITALOCEAN)
    if err != nil {
        panic(err)
    }

    ctx := context.TODO()

    serverResp, err := p.CreateServer(ctx, "test-server")
    if err != nil {
        panic(err)
    }

    subDomain := serverResp.Name + "-" + strconv.Itoa(serverResp.ServerID.(int)) + "." + "instances"

    dnsResp, err := p.CreateDNSRecord(ctx, subDomain, serverResp.ServerIP)
    if err != nil {
        panic(err)
    }

    fmt.Println("Sleeping for 120 seconds")
    time.Sleep(120 * time.Second)

    err2 := p.RemoveServer(ctx, serverResp.ServerID)
    if err2 != nil {
        panic(err2)
    }

    err3 := p.RemoveDNSRecord(ctx, dnsResp.SubDomainID)
    if err3 != nil {
        panic(err3)
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
