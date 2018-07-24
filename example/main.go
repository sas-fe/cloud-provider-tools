package main

import (
	"fmt"

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

	resp, err := p.CreateServer(
		"test-ondemand-po",
		common.ServerRegion("nyc1"),
		common.ServerSize("s-1vcpu-1gb"),
		common.ServerScript(startupScript),
		common.ServerTags([]string{"OnDemand"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	// fmt.Println("Sleeping for 120 seconds")
	// time.Sleep(120 * time.Second)

	// // err2 := p.RemoveServer(resp.ServerID, resp.SubDomainID)
	// err2 := p.RemoveServer(103160684, 50871058)
	// if err2 != nil {
	// 	panic(err2)
	// }
}
