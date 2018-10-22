package gce

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sas-fe/cloud-provider-tools/common"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
)

// Provider implements common.CloudProvider
type Provider struct {
	projectID    string
	computeSvc   *compute.Service
	containerSvc *container.ProjectsZonesClustersService
	dnsSvc       *dns.Service
	domain       string
	dnsZone      string
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

	svc, err := container.New(oauthClient)
	if err != nil {
		return nil, err
	}
	containerSvc := container.NewProjectsZonesClustersService(svc)

	dnsSvc, err := dns.New(oauthClient)
	if err != nil {
		return nil, err
	}

	return &Provider{projectID, computeSvc, containerSvc, dnsSvc, domain, dnsZone}, nil
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

// CreateServerGroup unimplemented for GCE
func (p *Provider) CreateServerGroup(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerGroupResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveServerGroup unimplemented for GCE
func (p *Provider) RemoveServerGroup(ctx context.Context, group *common.CreateServerGroupResponse) error {
	return errors.New("Unimplemented")
}

// CreateK8s creates a new cluster on GCE
func (p *Provider) CreateK8s(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateK8sResponse, error) {
	s := &common.ServerInfo{
		Name: name,
	}

	for _, opt := range opts {
		opt.Set(s)
	}

	prefix := "projects/" + p.projectID
	zone := s.Region
	region := zone[:len(zone)-2]
	machineType := s.Size
	initialCount := int64(3)
	autoScaling := &container.NodePoolAutoscaling{}
	version := s.K8sVersion

	if s.AutoScale != nil {
		initialCount = s.AutoScale.MinNodes
		autoScaling = &container.NodePoolAutoscaling{
			Enabled:      s.AutoScale.Enabled,
			MinNodeCount: s.AutoScale.MinNodes,
			MaxNodeCount: s.AutoScale.MaxNodes,
		}
	}

	cluster := &container.Cluster{
		Name: name,
		MasterAuth: &container.MasterAuth{
			Username: "admin",
			ClientCertificateConfig: &container.ClientCertificateConfig{
				IssueClientCertificate: true,
			},
		},
		LoggingService:    "logging.googleapis.com",
		MonitoringService: "monitoring.googleapis.com",
		Network:           prefix + "/global/networks/default",
		AddonsConfig: &container.AddonsConfig{
			HttpLoadBalancing: &container.HttpLoadBalancing{},
			KubernetesDashboard: &container.KubernetesDashboard{
				Disabled: true,
			},
		},
		Subnetwork: prefix + "/regions/" + region + "/subnetworks/default",
		NodePools: []*container.NodePool{
			&container.NodePool{
				Name: "default-pool",
				Config: &container.NodeConfig{
					MachineType: machineType,
					DiskSizeGb:  100,
					OauthScopes: []string{
						"https://www.googleapis.com/auth/devstorage.read_only",
						"https://www.googleapis.com/auth/logging.write",
						"https://www.googleapis.com/auth/monitoring",
						"https://www.googleapis.com/auth/servicecontrol",
						"https://www.googleapis.com/auth/service.management.readonly",
						"https://www.googleapis.com/auth/trace.append",
					},
					ImageType: "COS",
					// DiskType:  "pd-standard",
				},
				InitialNodeCount: initialCount,
				Autoscaling:      autoScaling,
				Management: &container.NodeManagement{
					AutoUpgrade: true,
					AutoRepair:  true,
				},
				Version: version,
			},
		},
		LegacyAbac: &container.LegacyAbac{
			Enabled: true,
		},
		InitialClusterVersion: version,
		Location:              zone,
	}

	_, err := p.containerSvc.Create(
		p.projectID,
		zone,
		&container.CreateClusterRequest{Cluster: cluster},
	).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	time.Sleep(120 * time.Second)

	var endpointIP string
	var credentials *common.ClusterCredentials
	ready := false
	for !ready {
		cls, err := p.containerSvc.Get(p.projectID, zone, name).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		if cls.Status == "RUNNING" {
			ready = true
			endpointIP = cls.Endpoint
			credentials = &common.ClusterCredentials{
				Username:    cls.MasterAuth.Username,
				Password:    cls.MasterAuth.Password,
				Certificate: cls.MasterAuth.ClusterCaCertificate,
			}
		} else {
			time.Sleep(15 * time.Second)
		}
	}

	return &common.CreateK8sResponse{
		Name:          name,
		ClusterID:     name,
		ClusterRegion: zone,
		EndpointIP:    endpointIP,
		EndpointPort:  "443",
		Credentials:   credentials,
	}, nil
}

// RemoveK8s removes a cluster on GCE
func (p *Provider) RemoveK8s(ctx context.Context, k8s *common.CreateK8sResponse) error {
	_, err := p.containerSvc.Delete(p.projectID, k8s.ClusterRegion, k8s.Name).Context(ctx).Do()
	if err != nil {
		return err
	}
	return nil
}

// CreateStaticIP creates a static IP on GCE
func (p *Provider) CreateStaticIP(ctx context.Context, name string, ipType common.StaticIPType) (*common.CreateStaticIPResponse, error) {
	addr := ""

	switch ipType {
	case common.GLOBAL:
		address := &compute.Address{
			Name:      name,
			IpVersion: "IPV4",
		}

		_, err := p.computeSvc.GlobalAddresses.Insert(p.projectID, address).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		time.Sleep(15 * time.Second)

		ready := false
		for !ready {
			resp, err := p.computeSvc.GlobalAddresses.Get(p.projectID, name).Context(ctx).Do()
			if err != nil {
				return nil, err
			}

			if len(resp.Address) > 0 {
				ready = true
				addr = resp.Address
			} else {
				time.Sleep(15 * time.Second)
			}
		}
	case common.REGIONAL:
		address := &compute.Address{
			Name: name,
		}
		region := "us-east1"

		_, err := p.computeSvc.Addresses.Insert(p.projectID, region, address).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		time.Sleep(15 * time.Second)
		ready := false
		for !ready {
			resp, err := p.computeSvc.Addresses.Get(p.projectID, region, name).Context(ctx).Do()
			if err != nil {
				return nil, err
			}

			if len(resp.Address) > 0 {
				ready = true
				addr = resp.Address
			} else {
				time.Sleep(15 * time.Second)
			}
		}
	default:
		return nil, fmt.Errorf("Static IP Type: %v is not supported", ipType)
	}

	return &common.CreateStaticIPResponse{
		Name:     name,
		StaticIP: addr,
		Type:     ipType,
	}, nil
}

// RemoveStaticIP removes a global static IP on GCE
func (p *Provider) RemoveStaticIP(ctx context.Context, staticIP *common.CreateStaticIPResponse) error {
	switch ipType := staticIP.Type; ipType {
	case common.GLOBAL:
		_, err := p.computeSvc.GlobalAddresses.Delete(p.projectID, staticIP.Name).Context(ctx).Do()
		if err != nil {
			return err
		}
	case common.REGIONAL:
		region := "us-east1"
		_, err := p.computeSvc.Addresses.Delete(p.projectID, region, staticIP.Name).Context(ctx).Do()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Static IP Type: %v is not supported", ipType)
	}

	return nil
}
