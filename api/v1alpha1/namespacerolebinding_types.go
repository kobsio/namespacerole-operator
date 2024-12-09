package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceRoleBindingSpec defines the desired state of NamespaceRoleBinding
type NamespaceRoleBindingSpec struct {
	// RoleRef is a reference to a NamespaceRole, which is used to create all the
	// ClusterRoleBindings and RoleBindings. These are created based on the status
	// field of the NamespaceRole.
	RoleRef  NamespaceRoleBindingSpecRoleRef `json:"roleRef"`
	Subjects []rbacv1.Subject                `json:"subjects"`
}

type NamespaceRoleBindingSpecRoleRef struct {
	// Name is the name of the NamespaceRole, which should be used by the
	// NamespaceRoleBinding.
	Name string `json:"name"`
}

// NamespaceRoleBindingStatus defines the observed state of NamespaceRoleBinding
type NamespaceRoleBindingStatus struct {
	// The label selector to get all ClusterRoleBindings / RoleBindings created by
	// the operator.
	Selector string `json:"selector,omitempty"`
	// ClusterRoleBindings is a list of ClusterRoleBindings which were created by
	// the operator.
	ClusterRoleBindings []NamespaceRoleStatusRoleBinding `json:"clusterRoleBindings,omitempty"`
	// RoleBinding is a list of RoleBindings which were created by the operator.
	RoleBindings []NamespaceRoleStatusRoleBinding `json:"roleBindings,omitempty"`
}

type NamespaceRoleStatusRoleBinding struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// NamespaceRoleBinding is the Schema for the namespacerolebindings API
// +kubebuilder:printcolumn:name="NamespaceRole",type=string,JSONPath=`.spec.roleRef.name`,description="The NamespaceRole used by the NamespaceRoleBinding"
// +kubebuilder:printcolumn:name="Selector",type=string,JSONPath=`.status.selector`,description="Selector to get all ClusterRoleBindings / RoleBindings created by the operator"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time when this NamespaceRoleBinding was created"
type NamespaceRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceRoleBindingSpec   `json:"spec,omitempty"`
	Status NamespaceRoleBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespaceRoleBindingList contains a list of NamespaceRoleBinding
type NamespaceRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceRoleBinding{}, &NamespaceRoleBindingList{})
}
