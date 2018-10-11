// Package digitalocean implements methods to create servers on DigitalOcean
package digitalocean

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sas-fe/cloud-provider-tools/common"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	AccessToken string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func clientFromToken(DOToken string) *godo.Client {
	oauthClient := oauth2.NewClient(oauth2.NoContext, &tokenSource{DOToken})
	return godo.NewClient(oauthClient)
}

// Provider implements common.CloudProvider
type Provider struct {
	client *godo.Client
	domain string
}

// NewProvider returns a new Provider instance
func NewProvider(DOToken string, domain string) *Provider {
	return &Provider{clientFromToken(DOToken), domain}
}

// CreateServer creates a droplet on DigitalOcean
// TODO reimplement waiting for IP using a ticker
func (p *Provider) CreateServer(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerResponse, error) {
	var dropletID int
	var dropletIP string

	s := &common.ServerInfo{
		Name: name,
	}

	for _, opt := range opts {
		opt.Set(s)
	}

	var imageIDStr string
	if len(s.Image) == 0 {
		imageIDStr = os.Getenv("DO_IMAGE_ID")
	} else {
		imageIDStr = s.Image
	}

	imageID, err := strconv.Atoi(imageIDStr)

	var image godo.DropletCreateImage
	if len(imageIDStr) == 0 || err != nil {
		image = godo.DropletCreateImage{
			Slug: "docker-16-04",
		}
	} else {
		image = godo.DropletCreateImage{
			ID: imageID,
		}
	}

	dropletRequest := &godo.DropletCreateRequest{
		Name:     s.Name,
		Region:   s.Region,
		Size:     s.Size,
		Image:    image,
		UserData: s.UserData,
		IPv6:     false,
		Tags:     s.Tags,
	}

	fmt.Println("Creating Droplet...")
	droplet, _, err := p.client.Droplets.Create(ctx, dropletRequest)
	if err != nil {
		return nil, err
	}
	dropletID = droplet.ID

	fmt.Println(dropletID)

	time.Sleep(60 * time.Second)

	ready := false
	for !ready {
		droplet, _, err := p.client.Droplets.Get(ctx, dropletID)
		if err != nil {
			return nil, err
		}

		if droplet.Status == "active" {
			fmt.Println("Droplet Created")
			ready = true
			dropletIP = droplet.Networks.V4[0].IPAddress
		} else {
			time.Sleep(15 * time.Second)
		}
	}

	fmt.Println(dropletIP)

	return &common.CreateServerResponse{
		Name:         name,
		ServerID:     dropletID,
		ServerRegion: s.Region,
		ServerIP:     dropletIP,
	}, nil
}

// CreateDNSRecord creates a DNS A Record on DigitalOcean
func (p *Provider) CreateDNSRecord(ctx context.Context, subDomain string, IP string) (*common.CreateDNSRecordResponse, error) {
	domainRequest := &godo.DomainRecordEditRequest{
		Type: "A",
		Name: subDomain,
		Data: IP,
	}
	domainRecord, _, err := p.client.Domains.CreateRecord(ctx, p.domain, domainRequest)
	if err != nil {
		return nil, err
	}

	return &common.CreateDNSRecordResponse{
		SubDomain:   subDomain,
		SubDomainID: domainRecord.ID,
	}, nil
}

// RemoveServer removes a droplet on DigitalOcean
func (p *Provider) RemoveServer(ctx context.Context, server *common.CreateServerResponse) error {
	intServerID, ok := server.ServerID.(int)
	if !ok {
		return fmt.Errorf("%v is not an int", server.ServerID)
	}

	fmt.Println("Deleting droplet...")
	_, err := p.client.Droplets.Delete(ctx, intServerID)
	if err != nil {
		return err
	}
	fmt.Println("Done")

	return nil
}

// RemoveDNSRecord removes a DNS A Record from DigitalOcean
func (p *Provider) RemoveDNSRecord(ctx context.Context, subDomain *common.CreateDNSRecordResponse) error {
	intSubDomainID, ok := subDomain.SubDomainID.(int)
	if !ok {
		return fmt.Errorf("%v is not an int", subDomain.SubDomainID)
	}

	fmt.Println("Deleting Domain Record...")
	_, err := p.client.Domains.DeleteRecord(ctx, p.domain, intSubDomainID)
	if err != nil {
		return err
	}
	fmt.Println("Done")

	return nil
}

// CreateServerGroup unimplemented for DigitalOcean
func (p *Provider) CreateServerGroup(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateServerGroupResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveServerGroup unimplemented for DigitalOcean
func (p *Provider) RemoveServerGroup(ctx context.Context, group *common.CreateServerGroupResponse) error {
	return errors.New("Unimplemented")
}

// CreateK8s unimplemented for DigitalOcean
func (p *Provider) CreateK8s(ctx context.Context, name string, opts ...common.ServerOption) (*common.CreateK8sResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveK8s unimplemented for DigitalOcean
func (p *Provider) RemoveK8s(ctx context.Context, k8s *common.CreateK8sResponse) error {
	return errors.New("Unimplemented")
}

// CreateStaticIP unimplemented for DigitalOcean
func (p *Provider) CreateStaticIP(ctx context.Context, name string) (*common.CreateStaticIPResponse, error) {
	return nil, errors.New("Unimplemented")
}

// RemoveStaticIP unimplemented for DigitalOcean
func (p *Provider) RemoveStaticIP(ctx context.Context, staticIP *common.CreateStaticIPResponse) error {
	return errors.New("Unimplemented")
}
