package gce

import (
	"context"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/common"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
)

// Provider implements common.CloudProvider
type Provider struct {
	projectID  string
	computeSvc *compute.Service
	dnsSvc     *dns.Service
	domain     string
	dnsZone    string
}

// NewProvider returns a new Provider instance
func NewProvider(projectID string, domain string, dnsZone string) (*Provider, error) {
	oauthClient, err := google.DefaultClient(oauth2.NoContext, compute.CloudPlatformScope, dns.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	computeSvc, err := compute.New(oauthClient)
	if err != nil {
		return nil, err
	}

	dnsSvc, err := dns.New(oauthClient)
	if err != nil {
		return nil, err
	}

	return &Provider{projectID, computeSvc, dnsSvc, domain, dnsZone}, nil
}

func (p *Provider) firewallsPreflight(prefix string) error {
	firewallHTTP := &compute.Firewall{
		Name:         "default-allow-http",
		SourceRanges: []string{"0.0.0.0/0"},
		Network:      prefix + "/global/networks/default",
		TargetTags:   []string{"http-server"},
		Allowed: []*compute.FirewallAllowed{
			&compute.FirewallAllowed{
				IPProtocol: "tcp",
				Ports:      []string{"80"},
			},
		},
	}

	firewallHTTPS := &compute.Firewall{
		Name:         "default-allow-https",
		SourceRanges: []string{"0.0.0.0/0"},
		Network:      prefix + "/global/networks/default",
		TargetTags:   []string{"https-server"},
		Allowed: []*compute.FirewallAllowed{
			&compute.FirewallAllowed{
				IPProtocol: "tcp",
				Ports:      []string{"443"},
			},
		},
	}

	_, err := p.computeSvc.Firewalls.Update(p.projectID, "default-allow-http", firewallHTTP).Do()
	if err != nil {
		return err
	}
	_, err = p.computeSvc.Firewalls.Update(p.projectID, "default-allow-https", firewallHTTPS).Do()
	if err != nil {
		return err
	}

	return nil
}

// CreateServer creates a droplet on GCP
// TODO reimplement waiting for IP using a ticker
func (p *Provider) CreateServer(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerResponse, error) {
	s := &common.ServerInfo{
		Name: name,
	}

	for _, opt := range opts {
		opt.Set(s)
	}

	prefix := "https://www.googleapis.com/compute/v1/projects/" + p.projectID
	zone := s.Region
	machineType := s.Size

	// p.firewallsPreflight(prefix)

	var imageURL string
	if len(s.Image) == 0 {
		imageURL = os.Getenv("GCP_SOURCE_IMAGE")
	} else {
		imageURL = s.Image
	}

	instance := &compute.Instance{
		Name:        name,
		MachineType: prefix + "/zones/" + zone + "/machineTypes/" + machineType,
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   "user-data",
					Value: &s.UserData,
				},
			},
		},
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    name + "-root-pd",
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: prefix + "/global/networks/default",
			},
		},
		Tags: &compute.Tags{
			Items: s.Tags,
		},
		ServiceAccounts: []*compute.ServiceAccount{
			&compute.ServiceAccount{
				Email: "default",
				Scopes: []string{
					compute.DevstorageReadOnlyScope,
				},
			},
		},
	}

	_, err := p.computeSvc.Instances.Insert(p.projectID, zone, instance).Do()
	if err != nil {
		return nil, err
	}

	time.Sleep(60 * time.Second)

	var serverIP string
	ready := false
	for !ready {
		ins, err := p.computeSvc.Instances.Get(p.projectID, zone, name).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		if ins.Status == "RUNNING" {
			ready = true
			serverIP = ins.NetworkInterfaces[0].AccessConfigs[0].NatIP
		} else {
			time.Sleep(15 * time.Second)
		}
	}

	return &common.CreateServerResponse{
		Name:         name,
		ServerID:     name,
		ServerRegion: zone,
		ServerIP:     serverIP,
	}, nil
}

// CreateDNSRecord creates a DNS A Record on GCP
func (p *Provider) CreateDNSRecord(ctx context.Context, subDomain string, IP string) (*common.CreateDNSRecordResponse, error) {
	rb := &dns.Change{
		Additions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Name:    subDomain + "." + p.domain + ".",
				Type:    "A",
				Ttl:     300,
				Rrdatas: []string{IP},
			},
		},
	}

	resp, err := p.dnsSvc.Changes.Create(p.projectID, p.dnsZone, rb).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return &common.CreateDNSRecordResponse{
		SubDomain:   subDomain,
		SubDomainID: resp.Id,
		SubDomainIP: IP,
	}, nil
}

// RemoveServer removes a droplet on GCP
func (p *Provider) RemoveServer(ctx context.Context, server *common.CreateServerResponse) error {
	_, err := p.computeSvc.Instances.Delete(p.projectID, server.ServerRegion, server.Name).Context(ctx).Do()
	if err != nil {
		return err
	}
	return nil
}

// RemoveDNSRecord removes a DNS A Record from GCP
func (p *Provider) RemoveDNSRecord(ctx context.Context, subDomain *common.CreateDNSRecordResponse) error {
	rb := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Name:    subDomain.SubDomain + "." + p.domain + ".",
				Type:    "A",
				Ttl:     300,
				Rrdatas: []string{subDomain.SubDomainIP},
			},
		},
	}

	_, err := p.dnsSvc.Changes.Create(p.projectID, p.dnsZone, rb).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}
