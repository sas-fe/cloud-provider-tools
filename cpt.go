// Package cpt provides common interfaces for creating on-demand servers with various providers
package cpt

import (
	"context"
	"fmt"
	"os"

	"github.com/sas-fe/cloud-provider-tools/common"
	"github.com/sas-fe/cloud-provider-tools/digitalocean"
)

// ProviderType provides an enum for cloud providers
type ProviderType int

const (
	// DIGITALOCEAN provider
	DIGITALOCEAN ProviderType = 0
	// AWS provider
	AWS ProviderType = 1
	// GCE provider
	GCE ProviderType = 2
	// AZURE provider
	AZURE ProviderType = 3
)

// CloudProvider implements methods for creating/removing serves
type CloudProvider interface {
	CreateServer(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerResponse, error)
	RemoveServer(ctx context.Context, serverID interface{}) error
	CreateDNSRecord(ctx context.Context, name string, IP string) (*common.CreateDNSRecordResponse, error)
	RemoveDNSRecord(ctx context.Context, subDomainID interface{}) error
}

var _ CloudProvider = (*digitalocean.Provider)(nil)

// NewCloudProvider returns a CloudProvider instance
func NewCloudProvider(pt ProviderType) (CloudProvider, error) {
	switch pt {
	case DIGITALOCEAN:
		fmt.Println("Using DigitalOcean")

		doToken := os.Getenv("DO_TOKEN")
		if len(doToken) == 0 {
			panic("$DO_TOKEN not set")
		}

		domain := os.Getenv("DOMAIN")
		if len(domain) == 0 {
			panic("$DOMAIN not set")
		}

		p := digitalocean.NewProvider(doToken, domain)
		return p, nil
	default:
		fmt.Println("Provider Not Implemented")
		return nil, fmt.Errorf("Provider Not Implemented")
	}
}
