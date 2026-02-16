package gateway

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	Group    = "foomo.org"
	Version  = "v1"
	Resource = "gateways"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=gw

// Gateway represents a service registration in the gateway.
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              Spec `json:"spec"`
}

// Service represents a backend service identifier.
// +kubebuilder:validation:MinLength=1
type Service string

// Path represents a URI path.
type Path string

// InternalAccessGroup represents an internal access group identifier.
type InternalAccessGroup string

// MimeType represents a MIME type.
type MimeType string

// Spec defines the desired state of a Gateway resource.
type Spec struct {
	Service     Service  `json:"service"`
	Sitemap     string   `json:"sitemap,omitempty"`
	AddToRobots string   `json:"addToRobots,omitempty"`
	Expose      []Expose `json:"expose,omitempty"`
}

// Expose defines an exposed path configuration.
type Expose struct {
	// A description of the exposition, to be show in gateway api
	Description string `json:"description,omitempty"`
	// Named application in cms
	CMSApp string `json:"cmsApp,omitempty"`
	// paths in urls to register for, lookup is automatically long to short
	Paths []Path `json:"paths,omitempty"`
	// Contentserver mimetypes
	CmsMimetypes []MimeType `json:"cmsMimetypes,omitempty"`
	// Access is only granted to these internal groups
	InternalAccessGroups []InternalAccessGroup `json:"internalAccessGroups,omitempty"`
	// Service is exposed at /foo request is /foo/bar, stripBasePath == true => requests will go to /bar and /foo will be set as x-base-path header
	StripBasePath bool `json:"stripBasePath,omitempty"`
}

// +kubebuilder:object:root=true

// List is a list of Gateway resources.
type List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Gateway `json:"items"`
}
