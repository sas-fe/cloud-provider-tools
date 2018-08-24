package common

// CreateServerResponse contains the response from server creation
type CreateServerResponse struct {
	Name     string
	ServerID interface{}
	ServerIP string
}

// CreateDNSRecordResponse contains the response from DNS record creation
type CreateDNSRecordResponse struct {
	SubDomain   string
	SubDomainID interface{}
}

// ServerInfo contains configuration information for the server
type ServerInfo struct {
	Name     string
	Size     string
	Region   string
	Image    string
	UserData string
	Tags     []string
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
