# cloud-provider-tools
[![GoDoc](https://godoc.org/github.com/sas-fe/cloud-provider-tools?status.svg)](https://godoc.org/github.com/sas-fe/cloud-provider-tools)

## Usage

```go
package main

import (
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

    serverResp, err := p.CreateServer("test-server")
    if err != nil {
        panic(err)
    }

    subDomain := serverResp.Name + "-" + strconv.Itoa(serverResp.ServerID.(int)) + "." + "instances"

    dnsResp, err := p.CreateDNSRecord(subDomain, serverResp.ServerIP)
    if err != nil {
        panic(err)
    }

    fmt.Println("Sleeping for 120 seconds")
    time.Sleep(120 * time.Second)

    err2 := p.RemoveServer(serverResp.ServerID)
    if err2 != nil {
        panic(err2)
    }

    err3 := p.RemoveDNSRecord(dnsResp.SubDomainID)
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
- Startup Script: e.g. `common.ServerScript("#!/bin/bash\necho 'Hello, World!'")`
- Tags: e.g. `common.ServerTags([]string{"OnDemand"})`
