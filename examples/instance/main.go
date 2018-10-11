package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/sas-fe/cloud-provider-tools"
	"github.com/sas-fe/cloud-provider-tools/common"
	yaml "gopkg.in/yaml.v2"
)

// Files contain template files
type Files struct {
	DockerCompose string
	NginxConf     string
}

// Values contain template values
type Values struct {
	Username       string `yaml:"username"`
	DockerUser     string `yaml:"dockerUser"`
	DockerPassword string `yaml:"dockerPassword"`

	Service struct {
		Name      string `yaml:"name"`
		ImageRepo string `yaml:"imageRepo"`
		ImageTag  string `yaml:"imageTag"`

		Ports []struct {
			Port     int    `yaml:"port"`
			Protocol string `yaml:"protocol"`
		} `yaml:"ports"`
	} `yaml:"service"`

	Ingress struct {
		Port    int  `yaml:"port"`
		Upgrade bool `yaml:"upgrade"`

		TLS struct {
			Bucket string `yaml:"bucket"`
			Path   string `yaml:"path"`
			Cert   string `yaml:"cert"`
			Key    string `yaml:"key"`
		} `yaml:"tls"`
	} `yaml:"ingress"`
}

// Config wraps both template files and values
type Config struct {
	Files  *Files
	Values *Values
}

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

func readValues(v *Values) error {
	yamlFile, err := ioutil.ReadFile("./examples/instance/values.yaml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, v)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	serverName := "face-recognition-" + srand(12)
	subDomain := serverName + "." + "instances"

	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}

	dockerUser := os.Getenv("DOCKER_USER")
	if len(dockerUser) == 0 {
		panic("$DOCKER_USER not set")
	}

	dockerPW := os.Getenv("DOCKER_PASSWORD")
	if len(dockerPW) == 0 {
		panic("$DOCKER_PASSWORD not set")
	}

	gcpImageURL := os.Getenv("GCP_SOURCE_IMAGE")
	if len(gcpImageURL) == 0 {
		gcpImageURL = "projects/ubuntu-os-cloud/global/images/ubuntu-1604-xenial-v20180912"
	}

	files := Files{}
	values := Values{}
	config := Config{&files, &values}

	err := readValues(&values)
	if err != nil {
		panic(err)
	}
	values.DockerUser = dockerUser
	values.DockerPassword = dockerPW

	t := template.Must(template.New("config").Funcs(sprig.TxtFuncMap()).ParseGlob("./examples/instance/templates/*"))

	composeBuf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(composeBuf, "docker-compose.yaml", config); err != nil {
		panic(err)
	}

	nginxBuf := new(bytes.Buffer)
	if err := t.ExecuteTemplate(nginxBuf, "nginx.conf", config); err != nil {
		panic(err)
	}

	files.DockerCompose = string(composeBuf.Bytes())
	files.NginxConf = string(nginxBuf.Bytes())

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
