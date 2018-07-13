package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/digitalocean"
)

func main() {
	doToken := os.Getenv("DO_TOKEN")
	if len(doToken) == 0 {
		panic("$DO_TOKEN not set")
	}

	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}

	p := digitalocean.NewProvider(doToken, domain)

	resp, err := p.CreateServer("test-ondemand")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	fmt.Println("Sleeping for 120 seconds")
	time.Sleep(120 * time.Second)

	err2 := p.RemoveServer(resp.ServerID, resp.SubDomainID)
	// err2 := p.RemoveServer(101309830, 49549772)
	if err2 != nil {
		panic(err2)
	}
}
