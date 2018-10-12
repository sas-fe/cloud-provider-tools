package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	cpt "github.com/sas-fe/cloud-provider-tools"
	"github.com/sas-fe/cloud-provider-tools/common"

	"k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
)

const alphaNumericsLower = "abcdefghijklmnopqrstuvwxyz0123456789"

var chartsDir = flag.String("charts-dir", "./", "Root directory of helm charts")
var valuesDir = flag.String("values-dir", "./", "Root directory of values overrides")

type chart struct {
	Name        string
	Path        string
	ValuesFiles []string
	Values      []string
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func srand(length int) string {
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = alphaNumericsLower[rand.Intn(len(alphaNumericsLower))]
	}
	return string(buf)
}

func newK8sConnection(k8sResp *common.CreateK8sResponse, namespace string, initTiller bool) (*helm.Client, *kubernetes.Clientset, string, error) {
	CAData, err := base64.StdEncoding.DecodeString(k8sResp.Credentials.Certificate)
	if err != nil {
		return nil, nil, "", err
	}

	scheme := "https://"

	k8sConfig := &rest.Config{
		Host:     scheme + k8sResp.EndpointIP,
		Username: k8sResp.Credentials.Username,
		Password: k8sResp.Credentials.Password,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: CAData,
		},
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, nil, "", err
	}

	svcResp, err := k8sClient.CoreV1().Services("kube-system").Get("kube-dns", metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}

	if initTiller {
		fmt.Println("Installing tiller")

		sa := &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tiller",
			},
		}
		_, err := k8sClient.CoreV1().ServiceAccounts(namespace).Create(sa)
		if err != nil {
			return nil, nil, "", err
		}

		crb := &rbacv1beta1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tiller",
			},
			RoleRef: rbacv1beta1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []rbacv1beta1.Subject{
				rbacv1beta1.Subject{
					Kind:      "ServiceAccount",
					Name:      "tiller",
					Namespace: namespace,
				},
			},
		}
		_, err = k8sClient.RbacV1beta1().ClusterRoleBindings().Create(crb)
		if err != nil {
			return nil, nil, "", err
		}

		err = installTiller(namespace, k8sClient)
		if err != nil {
			return nil, nil, "", err
		}
	}

	tunnel, err := portforwarder.New(namespace, k8sClient, k8sConfig)
	if err != nil {
		return nil, nil, "", err
	}

	tillerHost := fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	helmClient := helm.NewClient(helm.Host(tillerHost))

	return helmClient, k8sClient, svcResp.Spec.ClusterIP, nil
}

func installTiller(namespace string, client kubernetes.Interface) error {
	opts := &installer.Options{
		Namespace:                    namespace,
		ImageSpec:                    "gcr.io/kubernetes-helm/tiller:v2.9.1",
		ServiceAccount:               "tiller",
		AutoMountServiceAccountToken: true,
	}
	if err := installer.Install(client, opts); err != nil {
		return err
	}

	ready := watchTillerUntilReady(namespace, client, 120)
	if !ready {
		return errors.New("Tiller not ready")
	}

	return nil
}

func main() {
	flag.Parse()

	startTime := time.Now()

	clusterName := "svi-" + srand(12)
	subDomain := clusterName + "." + "instances"

	domain := os.Getenv("DOMAIN")
	if len(domain) == 0 {
		panic("$DOMAIN not set")
	}
	p, err := cpt.NewCloudProvider(cpt.GCE)
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()

	fmt.Println("Acquiring global static IP")
	ipResp, err := p.CreateStaticIP(ctx, clusterName)
	if err != nil {
		panic(err)
	}
	fmt.Println(ipResp)

	fmt.Println("Creating DNS record")
	dnsResp, err := p.CreateDNSRecord(ctx, subDomain, ipResp.StaticIP)
	if err != nil {
		panic(err)
	}
	fmt.Println(dnsResp)

	fmt.Println("Creating Cluster")
	k8sResp, err := p.CreateK8s(
		ctx,
		clusterName,
		common.ServerRegion("us-east1-c"),
		common.ServerSize("n1-standard-4"),
		common.AutoScale(&common.AutoScaleOpt{
			Enabled:  true,
			MinNodes: 3,
			MaxNodes: 10,
		}),
		common.K8sVersion("1.10.6-gke.6"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(k8sResp)
	fmt.Println(k8sResp.Credentials)

	// k8sResp := &common.CreateK8sResponse{
	// 	EndpointIP: "35.185.44.172",
	// 	Credentials: &common.ClusterCredentials{
	// 		Username:    "admin",
	// 		Password:    "44qfR0l6Pk3Cqrq1",
	// 		Certificate: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURDekNDQWZPZ0F3SUJBZ0lRQmN6d2FvUVcvSjRPb1dqVG1RUW1VakFOQmdrcWhraUc5dzBCQVFzRkFEQXYKTVMwd0t3WURWUVFERXlRNE56a3lOMlEzTUMxaFpXRXlMVFJpT0dVdFlURmhPUzA0WW1JNE5qQmxZbVkxT0dVdwpIaGNOTVRneE1ERXdNVFl4TXpBMldoY05Nak14TURBNU1UY3hNekEyV2pBdk1TMHdLd1lEVlFRREV5UTROemt5Ck4yUTNNQzFoWldFeUxUUmlPR1V0WVRGaE9TMDRZbUk0TmpCbFltWTFPR1V3Z2dFaU1BMEdDU3FHU0liM0RRRUIKQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURqQkZHV2FlY3JTVzRNUDlCTktvNy9UNUI1L2RlTVNtR0JSeG9EaUcvMQpqUytKSjlINFhUdE1wUDZMVDZ6a3AyNDIvVDZNeWR6QlZxZ2xoQ2J1REhnN3pxYnZDMWhIVmFIMk9TbXpLY3E5CjVvS21hUTVsdlc3RFNDZ2l6S0gvN3dtTG5UOVk5ZnMvMWozdkIyS3NLdlJHOWNUNi9wYzc0eDJmTDFSNjFvTDEKQ2RVOG53a01DNHhxZzlCN0tFbWxmUlVWaTdpM2JybUw4b0Fxb21jLzVwbllLQ2g3bkhVU213Rmd4SkJnTlZXMwpYTktyTjEyR0loRW52Z3ZzMUtIeU41cUhHSkc3RkUxMUQrcTJGa3RZMUhiNEx1d2duOGF0SDIrdWRsU2VLSHZjCjZjRlgycG80S1NnbktMV2dOeTBhRTJpRFFBWGtFMXQxL3BoMzRQVWFQMlp2QWdNQkFBR2pJekFoTUE0R0ExVWQKRHdFQi93UUVBd0lDQkRBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFDUwpYNlFKVlh3eW1ZRmk0Q3pBUXptTTlYbjhqMkNsRW93Zm1UNkxZWUxtWlJsOEJyQnBYMmdIeTBtcTdLVkJUdTMxClU3ZEdtTlc0VzIxSEl6NDNwQ3lnSlkrcW9VSTN1Q1pRTzBNWFdhTUtVU3N6TUZSSDZTd3d2T0Y1ZVBlVlFQOFAKUnJmUndLeEV2N0NRa2UwNk8xVFhyWkhTam1WVFg0T1E1S0ZUbnFRcmozaHRINWVZZWtNUHFnbTB6VHhleFpLbAp5d3NoTG5hWXM1SHRESmNnRjFLbmo2WWtxa25sSmxvWEtGTzZwSnNEVUVxb08zWFYxUWVlMFRzWHB4MnR5UUNOCmwxWDVuYi9Jd210MUZiSlJlUTNwb0d2MHVFd1d6ekUwK21KRzNaUjZpUG5mOVprYkgzQzI2Z2lCOGVrYzUwNlEKb2c3ekJ6bEFVUHVpZ1d2R0FaTE0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
	// 	},
	// }

	tillerNamespace := "kube-system"
	initTiller := true

	fmt.Println("Creating Helm client")
	helmClient, k8sClient, kubednsIP, err := newK8sConnection(k8sResp, tillerNamespace, initTiller)
	if err != nil {
		fmt.Printf("Error creating helm client: %v\n", err)
		panic(err)
	}
	fmt.Printf("KUBE-DNS IP: %v\n", kubednsIP)

	namespace := "default"
	wait := true
	timeout := int64(3600)
	dryrun := false
	charts := []chart{
		chart{
			Name:        "openldap",
			Path:        path.Join(*chartsDir, "InfrastructureServices"),
			ValuesFiles: []string{path.Join(*valuesDir, "openldap.yaml")},
			Values:      []string{},
		},
		chart{
			Name:        "infrastructure",
			Path:        path.Join(*chartsDir, "ViyaInfrastructureServices"),
			ValuesFiles: []string{path.Join(*valuesDir, "infrastructure.yaml")},
			Values:      []string{},
		},
		chart{
			Name:        "petrichor",
			Path:        path.Join(*chartsDir, "ViyaPetrichorServices"),
			ValuesFiles: []string{path.Join(*valuesDir, "petrichor.yaml")},
			Values:      []string{fmt.Sprintf("k8s.kubednsIP=%s", kubednsIP)},
		},
		chart{
			Name:        "svi-general",
			Path:        path.Join(*chartsDir, "SVIGeneralServices"),
			ValuesFiles: []string{path.Join(*valuesDir, "svi-general.yaml")},
			Values:      []string{fmt.Sprintf("k8s.kubednsIP=%s", kubednsIP)},
		},
		chart{
			Name:        "visual-investigator",
			Path:        path.Join(*chartsDir, "VisualInvestigator"),
			ValuesFiles: []string{path.Join(*valuesDir, "visual-investigator.yaml")},
			Values:      []string{fmt.Sprintf("k8s.kubednsIP=%s", kubednsIP)},
		},
		chart{
			Name:        "ingress",
			Path:        path.Join(*chartsDir, "GCPIngress"),
			ValuesFiles: []string{},
			Values: []string{
				fmt.Sprintf("system.staticIP=%s", ipResp.Name),
				fmt.Sprintf("system.host=%s", subDomain+"."+domain),
			},
		},
	}

	for _, c := range charts {
		fmt.Printf("Preparing release: %v\n", c.Name)
		rawVals, err := vals(c.ValuesFiles, c.Values)
		if err != nil {
			panic(err)
		}
		opts := []helm.InstallOption{
			helm.ReleaseName(c.Name),
			helm.InstallWait(wait),
			helm.InstallTimeout(timeout),
			helm.ValueOverrides(rawVals),
			helm.InstallDryRun(dryrun),
		}
		fmt.Printf("Installing release: %v\n", c.Name)
		_, err = helmClient.InstallRelease(c.Path, namespace, opts...)
		if err != nil {
			panic(err)
		}
	}

	listResp, err := helmClient.ListReleases()
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nInstalled Releases:\n")
	for _, r := range listResp.Releases {
		fmt.Println(r.Name)
	}

	up := waitforIngress(k8sClient, namespace, "ingress-svi-ingress", timeout)
	if !up {
		panic("Ingress creation timed out")
	}

	jobChart := chart{
		Name:        "init-data",
		Path:        path.Join(*chartsDir, "SVIInitData"),
		ValuesFiles: []string{},
		Values:      []string{fmt.Sprintf("system.host=%s", subDomain+"."+domain)},
	}
	fmt.Printf("Preparing job: %v\n", jobChart.Name)
	rawVals, err := vals(jobChart.ValuesFiles, jobChart.Values)
	if err != nil {
		panic(err)
	}
	opts := []helm.InstallOption{
		helm.ReleaseName(jobChart.Name),
		helm.InstallWait(wait),
		helm.InstallTimeout(timeout),
		helm.ValueOverrides(rawVals),
		helm.InstallDryRun(dryrun),
	}
	fmt.Printf("Running job: %v\n", jobChart.Name)
	_, err = helmClient.InstallRelease(jobChart.Path, namespace, opts...)
	if err != nil {
		panic(err)
	}
	done := waitForJobs(k8sClient, namespace, timeout)
	if !done {
		panic("Timed out waiting for jobs to finish")
	}

	timeElapsed := time.Since(startTime)
	fmt.Printf("Done in %v\n", timeElapsed.Seconds())
}
