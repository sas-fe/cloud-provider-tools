package digitalocean

import (
	"context"
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

// CreateServer creates a droplet on digitalocean
// TODO refactor to use ServerInfo and ServerOptions
// TODO move image ID to env variable
func (p *Provider) CreateServer(name string, opts ...common.ServerOption) (*common.CreateResponse, error) {
	var dropletID int
	var dropletIP string

	ctx := context.TODO()

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
		UserData: s.Script,
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

	subDomain := name + "-" + strconv.Itoa(dropletID) + "." + "instances"
	domainRequest := &godo.DomainRecordEditRequest{
		Type: "A",
		Name: subDomain,
		Data: dropletIP,
	}
	domainRecord, _, err := p.client.Domains.CreateRecord(ctx, p.domain, domainRequest)
	if err != nil {
		return nil, err
	}

	return &common.CreateResponse{
		Name:        name,
		ServerID:    dropletID,
		SubDomain:   subDomain,
		SubDomainID: domainRecord.ID,
	}, nil
}

// RemoveServer removes a droplet on digitalocean
func (p *Provider) RemoveServer(serverID interface{}, subDomainID interface{}) error {
	intServerID, ok := serverID.(int)
	if !ok {
		return fmt.Errorf("%v is not an int", serverID)
	}

	intSubDomainID, ok := subDomainID.(int)
	if !ok {
		return fmt.Errorf("%v is not an int", subDomainID)
	}

	ctx := context.TODO()

	fmt.Println("Deleting Domain Record...")
	_, err := p.client.Domains.DeleteRecord(ctx, p.domain, intSubDomainID)
	if err != nil {
		return err
	}
	fmt.Println("Done")

	fmt.Println("Deleting droplet...")
	_, err = p.client.Droplets.Delete(ctx, intServerID)
	if err != nil {
		return err
	}
	fmt.Println("Done")

	return nil
}