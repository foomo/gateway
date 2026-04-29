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

// MimeType represents a MIME type.
type MimeType string

// InternalAccessGroup represents an internal access group identifier.
type InternalAccessGroup string

// Spec defines the desired state of a Gateway resource.
type Spec struct {
	Service       Service  `json:"service"`
	ErrorFrontend bool     `json:"errorFrontend,omitempty"`
	Sitemap       string   `json:"sitemap,omitempty"`
	AddToRobots   string   `json:"addToRobots,omitempty"`
	Expose        []Expose `json:"expose,omitempty"`
}

// Expose defines an exposed path configuration.
type Expose struct {
	// A description of the exposition, to be show in gateway api
	Description string `json:"description,omitempty"`
	// Contentserver mimetypes
	CmsMimetypes []MimeType `json:"cmsMimetypes,omitempty"`
	// InternalAccessGroups restricts access to the listed internal groups.
	InternalAccessGroups []InternalAccessGroup `json:"internalAccessGroups,omitempty"`
}

// +kubebuilder:object:root=true

// List is a list of Gateway resources.
type List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Gateway `json:"items"`
}
