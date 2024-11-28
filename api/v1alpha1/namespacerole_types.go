package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceRoleSpec defines the desired state of NamespaceRole
type NamespaceRoleSpec struct {
	// Namespaces is a list of namespace the Roles should be created in. If the
	// list only contains one value, which is equal to "*", a ClusterRole instead
	// of a Role will be created.
	Namespaces []string            `json:"namespaces"`
	Rules      []rbacv1.PolicyRule `json:"rules"`
}

// NamespaceRoleStatus defines the observed state of NamespaceRole
type NamespaceRoleStatus struct {
	// ClusterRoles is a list of ClusterRoles which were created by the operator.
	ClusterRoles []NamespaceRoleStatusRole `json:"clusterRoles,omitempty"`
	// Roles is a list of Roles which were created by the operator.
	Roles []NamespaceRoleStatusRole `json:"roles,omitempty"`
}

type NamespaceRoleStatusRole struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// NamespaceRole is the Schema for the namespaceroles API
type NamespaceRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceRoleSpec   `json:"spec,omitempty"`
	Status NamespaceRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespaceRoleList contains a list of NamespaceRole
type NamespaceRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceRole{}, &NamespaceRoleList{})
}
