package main

import (
	"fmt"
	"strconv"

	"github.com/sas-fe/cloud-provider-tools"
	"github.com/sas-fe/cloud-provider-tools/common"
)

func main() {
	p, err := cpt.NewCloudProvider(cpt.DIGITALOCEAN)
	if err != nil {
		panic(err)
	}

	startupScript := `#!/bin/bash
	docker run -p 9999:9999 -d --rm sasfe/sas4c:python-only`

	serverResp, err := p.CreateServer(
		"test-ondemand",
		common.ServerRegion("nyc1"),
		common.ServerSize("s-1vcpu-1gb"),
		common.ServerScript(startupScript),
		common.ServerTags([]string{"OnDemand"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(serverResp)

	subDomain := serverResp.Name + "-" + strconv.Itoa(serverResp.ServerID.(int)) + "." + "instances"

	dnsResp, err := p.CreateDNSRecord(subDomain, serverResp.ServerIP)
	if err != nil {
		panic(err)
	}

	fmt.Println(dnsResp)

	// fmt.Println("Sleeping for 120 seconds")
	// time.Sleep(120 * time.Second)

	// err2 := p.RemoveServer(serverResp.ServerID)
	// // err2 := p.RemoveServer(101309830)
	// if err2 != nil {
	// 	panic(err2)
	// }

	// err3 := p.RemoveDNSRecord(dnsResp.SubDomainID)
	// // err2 := p.RemoveDNSRecord(49549772)
	// if err3 != nil {
	// 	panic(err3)
	// }
}
