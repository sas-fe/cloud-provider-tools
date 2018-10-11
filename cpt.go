// Package cpt provides common interfaces for creating on-demand servers with various providers
package cpt

import (
	"context"
	"fmt"
	"os"

	"github.com/sas-fe/cloud-provider-tools/common"
	"github.com/sas-fe/cloud-provider-tools/digitalocean"
	"github.com/sas-fe/cloud-provider-tools/gce"
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
	RemoveServer(ctx context.Context, server *common.CreateServerResponse) error

	CreateServerGroup(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerGroupResponse, error)
	RemoveServerGroup(ctx context.Context, group *common.CreateServerGroupResponse) error

	CreateK8s(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateK8sResponse, error)
	RemoveK8s(ctx context.Context, k8s *common.CreateK8sResponse) error

	CreateDNSRecord(ctx context.Context, name string, IP string) (*common.CreateDNSRecordResponse, error)
	RemoveDNSRecord(ctx context.Context, subDomain *common.CreateDNSRecordResponse) error

	CreateStaticIP(ctx context.Context, name string) (*common.CreateStaticIPResponse, error)
	RemoveStaticIP(ctx context.Context, staticIP *common.CreateStaticIPResponse) error
}

var _ CloudProvider = (*digitalocean.Provider)(nil)
var _ CloudProvider = (*gce.Provider)(nil)

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
	case GCE:
		fmt.Println("Using GCE")

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

		p, err := gce.NewProvider(projectID, domain, dnsZone)
		if err != nil {
			return nil, err
		}

		return p, nil
	default:
		fmt.Println("Provider Not Implemented")
		return nil, fmt.Errorf("Provider Not Implemented")
	}
}
