package common

// CreateServerResponse contains the response from server creation
type CreateServerResponse struct {
	Name         string
	ServerID     interface{}
	ServerRegion string
	ServerIP     string
}

// CreateDNSRecordResponse contains the response from DNS record creation
type CreateDNSRecordResponse struct {
	SubDomain   string
	SubDomainID interface{}
	SubDomainIP string
}

// CreateServerGroupResponse contains the reponse from creating a server group
type CreateServerGroupResponse struct {
	Name              string
	ServerGroupID     interface{}
	ServerGroupRegion string
	LoadBalancerID    string
	LoadBalancerIP    string
}

// ClusterCredentials contain credentials for the k8s cluster
type ClusterCredentials struct {
	Username    string
	Password    string
	Certificate string
}

// CreateK8sResponse contains the response from K8s deployment
type CreateK8sResponse struct {
	Name          string
	ClusterID     interface{}
	ClusterRegion string
	EndpointIP    string
	EndpointPort  string
	Credentials   *ClusterCredentials
}

// StaticIPType enums the type of static IP
type StaticIPType int

const (
	// GLOBAL static IP
	GLOBAL StaticIPType = 0
	// REGIONAL static IP
	REGIONAL StaticIPType = 1
)

// CreateStaticIPResponse contains the response from creating a static IP
type CreateStaticIPResponse struct {
	Name     string
	StaticIP string
	Type     StaticIPType
}

// AutoScaleOpt contains fields for k8s autoscaling
type AutoScaleOpt struct {
	Enabled  bool
	MinNodes int64
	MaxNodes int64
}

// ServerInfo contains configuration information for the server
type ServerInfo struct {
	Name       string
	Size       string
	AutoScale  *AutoScaleOpt
	Region     string
	Image      string
	K8sVersion string
	UserData   string
	Tags       []string
}

// ServerOption configures a server for creation
type ServerOption interface {
	Set(*ServerInfo) error
}

// SizeServerOption configures the server size
type SizeServerOption struct {
	Size string
}

// Set sets the server size
func (o SizeServerOption) Set(s *ServerInfo) error {
	s.Size = o.Size
	return nil
}

// ServerSize returns a ServerOption that sets the size
func ServerSize(size string) ServerOption {
	return SizeServerOption{size}
}

// RegionServerOption configures the server region
type RegionServerOption struct {
	Region string
}

// Set sets the server region
func (o RegionServerOption) Set(s *ServerInfo) error {
	s.Region = o.Region
	return nil
}

// ServerRegion returns a ServerOption that sets the region
func ServerRegion(region string) ServerOption {
	return RegionServerOption{region}
}

// ImageServerOption configures the server image
type ImageServerOption struct {
	Image string
}

// Set sets the server image
func (o ImageServerOption) Set(s *ServerInfo) error {
	s.Image = o.Image
	return nil
}

// ServerImage returns a ServerOption that sets the image
func ServerImage(image string) ServerOption {
	return ImageServerOption{image}
}

// UserDataServerOption configures the server UserData with cloud-init
type UserDataServerOption struct {
	UserData string
}

// Set sets the server UserData
func (o UserDataServerOption) Set(s *ServerInfo) error {
	s.UserData = o.UserData
	return nil
}

// ServerUserData returns a ServerOption that sets the UserData
func ServerUserData(userdata string) ServerOption {
	return UserDataServerOption{userdata}
}

// TagsServerOption configures the server tags
type TagsServerOption struct {
	Tags []string
}

// Set sets the server tags
func (o TagsServerOption) Set(s *ServerInfo) error {
	s.Tags = o.Tags
	return nil
}

// ServerTags returns a ServerOption that sets the tags
func ServerTags(tags []string) ServerOption {
	return TagsServerOption{tags}
}

// AutoScaleServerOption configures the k8s autoscaling
type AutoScaleServerOption struct {
	AutoScale *AutoScaleOpt
}

// Set sets the k8s autoscaling
func (o AutoScaleServerOption) Set(s *ServerInfo) error {
	s.AutoScale = o.AutoScale
	return nil
}

// AutoScale returns a ServerOption that sets the k8s autoscaling
func AutoScale(opt *AutoScaleOpt) ServerOption {
	return AutoScaleServerOption{opt}
}

// K8sVersionServerOption configures the k8s version
type K8sVersionServerOption struct {
	Version string
}

// Set sets the k8s autoscaling
func (o K8sVersionServerOption) Set(s *ServerInfo) error {
	s.K8sVersion = o.Version
	return nil
}

// K8sVersion returns a ServerOption that sets the k8s version
func K8sVersion(version string) ServerOption {
	return K8sVersionServerOption{version}
}
