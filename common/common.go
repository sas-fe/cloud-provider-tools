package common

// CreateResponse contains the response from server creation
type CreateResponse struct {
	Name        string
	ServerID    interface{}
	SubDomain   string
	SubDomainID interface{}
}

// serverInfo contains configuration information for the server
type ServerInfo struct {
	name   string
	size   string
	region string
	image  string
}

// ServerOption configures a server for creation
type ServerOption interface {
	set(*ServerInfo) error
}

// SizeServerOption configures the server size
type SizeServerOption struct {
	Size string
}

func (o SizeServerOption) set(s *ServerInfo) error {
	s.size = o.Size
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

func (o RegionServerOption) set(s *ServerInfo) error {
	s.region = o.Region
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

func (o ImageServerOption) set(s *ServerInfo) error {
	s.image = o.Image
	return nil
}

// ServerImage returns a ServerOption that sets the image
func ServerImage(image string) ServerOption {
	return ImageServerOption{image}
}
