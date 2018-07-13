package main

import (
	"fmt"

	"github.com/sas-fe/cloud-provider-tools"
)

func main() {
	p, err := cpt.NewCloudProvider(cpt.DIGITALOCEAN)
	if err != nil {
		panic(err)
	}

	resp, err := p.CreateServer("test-ondemand")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	// fmt.Println("Sleeping for 120 seconds")
	// time.Sleep(120 * time.Second)

	// err2 := p.RemoveServer(resp.ServerID, resp.SubDomainID)
	// // err2 := p.RemoveServer(101333846, 49574524)
	// if err2 != nil {
	// 	panic(err2)
	// }
}
