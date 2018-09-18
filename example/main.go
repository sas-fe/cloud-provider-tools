package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/sas-fe/cloud-provider-tools"
	"github.com/sas-fe/cloud-provider-tools/common"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

const alphaNumericsLower = "abcdefghijklmnopqrstuvwxyz0123456789"

func srand(length int) string {
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = alphaNumericsLower[rand.Intn(len(alphaNumericsLower))]
	}
	return string(buf)
}

// Port exports the port number and type
type Port struct {
	Number int
	Type   string
}

// Service contains service config values
type Service struct {
	Name      string
	VolRoot   string
	Domain    string
	SubDomain string
	Image     string
	Ports     []Port
}

// Config contain template values
type Config struct {
	UserName      string
	UserHome      string
	Commands      []string
	DockerCompose string
	NginxConf     string
}

// Ingress contains config values for ingress service
type Ingress struct {
	ServicePort int
	Upgrade     bool
	Service     Service
}

func main() {
	username := "testuser"
	userhome := "/home/" + username
	serverName := "face-recognition-" + srand(12)
	subDomain := serverName + "." + "instances"

	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}

	gcpImageURL := os.Getenv("GCP_SOURCE_IMAGE")
	if len(gcpImageURL) == 0 {
		panic("$GCP_SOURCE_IMAGE not set")
	}

	dockerUser := os.Getenv("DOCKER_USER")
	if len(dockerUser) == 0 {
		panic("$DOCKER_USER not set")
	}

	dockerPW := os.Getenv("DOCKER_PASSWORD")
	if len(dockerPW) == 0 {
		panic("$DOCKER_PASSWORD not set")
	}

	dockerImage := os.Getenv("DOCKER_IMAGE")
	if len(dockerImage) == 0 {
		panic("$DOCKER_IMAGE not set")
	}

	t := template.Must(template.New("config").Funcs(sprig.TxtFuncMap()).ParseGlob("./example/templates/*"))

	service := Service{
		Name:      "face-recognition",
		VolRoot:   userhome,
		Domain:    domain,
		SubDomain: subDomain,
		Image:     dockerImage,
		Ports: []Port{
			Port{8080, "HTTP"},
			Port{1935, "TCP"},
			Port{10001, "TCP"},
		},
	}

	composeBuf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(composeBuf, "docker-compose.yaml", service); err != nil {
		panic(err)
	}

	nConf := Ingress{
		ServicePort: 8080,
		Upgrade:     true,
		Service:     service,
	}

	nginxBuf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(nginxBuf, "nginx.conf", nConf); err != nil {
		panic(err)
	}

	installDocker := fmt.Sprintf("curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh")
	installCompose := fmt.Sprintf("curl -L 'https://github.com/docker/compose/releases/download/1.11.2/docker-compose-Linux-x86_64' -o /usr/local/bin/docker-compose && chmod +x /usr/local/bin/docker-compose")
	dockerLogin := fmt.Sprintf("docker login --username=%s --password=%s", dockerUser, dockerPW)
	dockerPull := fmt.Sprintf("docker-compose -f /home/%s/docker-compose.yaml pull", username)
	dockerUp := fmt.Sprintf("runuser -l %s -c 'sudo docker-compose -f /home/%s/docker-compose.yaml up -d'", username, username)
	config := Config{
		UserName:      username,
		UserHome:      userhome,
		DockerCompose: string(composeBuf.Bytes()),
		NginxConf:     string(nginxBuf.Bytes()),
	}
	config.Commands = []string{}
	config.Commands = append(config.Commands, installDocker)
	config.Commands = append(config.Commands, installCompose)
	config.Commands = append(config.Commands, dockerLogin)
	config.Commands = append(config.Commands, dockerPull)
	config.Commands = append(config.Commands, dockerUp)

	configBuf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(configBuf, "cloud-config.yaml", config); err != nil {
		panic(err)
	}
	fmt.Println(configBuf.String())

	p, err := cpt.NewCloudProvider(cpt.GCE)
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()

	serverResp, err := p.CreateServer(
		ctx,
		serverName,
		common.ServerImage(gcpImageURL),
		common.ServerRegion("us-east1-c"),
		common.ServerSize("n1-highcpu-4"),
		common.ServerUserData(configBuf.String()),
		common.ServerTags([]string{"http-server", "https-server", "face-recognition"}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(serverResp)

	dnsResp, err := p.CreateDNSRecord(ctx, subDomain, serverResp.ServerIP)
	if err != nil {
		panic(err)
	}

	fmt.Println(dnsResp)

	// fmt.Println("Sleeping for 120 seconds")
	// time.Sleep(120 * time.Second)

	// err2 := p.RemoveServer(ctx, serverResp.ServerID)
	// // err2 := p.RemoveServer(ctx, 101309830)
	// if err2 != nil {
	// 	panic(err2)
	// }

	// err3 := p.RemoveDNSRecord(ctx, dnsResp.SubDomainID)
	// // err2 := p.RemoveDNSRecord(ctx, 49549772)
	// if err3 != nil {
	// 	panic(err3)
	// }
}
