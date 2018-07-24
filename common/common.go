package common

// CreateResponse contains the response from server creation
type CreateResponse struct {
	Name        string
	ServerID    interface{}
	SubDomain   string
	SubDomainID interface{}
}

// ServerInfo contains configuration information for the server
type ServerInfo struct {
	Name   string
	Size   string
	Region string
	Image  string
	Script string
	Tags   []string
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

// ScriptServerOption configures the server startup script
type ScriptServerOption struct {
	Script string
}

// Set sets the server startup script
func (o ScriptServerOption) Set(s *ServerInfo) error {
	s.Script = o.Script
	return nil
}

// ServerScript returns a ServerOption that sets the startup script
func ServerScript(script string) ServerOption {
	return ScriptServerOption{script}
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
