package controller

import (
	"context"
	"fmt"

	kobsiov1alpha1 "github.com/kobsio/namespacerole-operator/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	selectorLabelKeyNRB = "kobs.io/namespacerolebinding"
)

// NamespaceRoleBindingReconciler reconciles a NamespaceRoleBinding object
type NamespaceRoleBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kobs.io,resources=namespacerolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kobs.io,resources=namespacerolebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kobs.io,resources=namespacerolebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state. For more
// details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *NamespaceRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconcile NamespaceRoleBinding")

	namespaceRoleBinding := &kobsiov1alpha1.NamespaceRoleBinding{}
	err := r.Get(ctx, req.NamespacedName, namespaceRoleBinding)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after
			// reconcile request. Owned objects are automatically garbage
			// collected. For additional cleanup logic use finalizers. Return
			// and don't requeue
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get NamespaceRoleBinding")
		return ctrl.Result{}, err
	}

	namespaceRole := &kobsiov1alpha1.NamespaceRole{}
	if err := r.Get(ctx, types.NamespacedName{Name: namespaceRoleBinding.Spec.RoleRef.Name}, namespaceRole); err != nil {
		log.Error(err, "Failed to get NamespaceRole", "NamespaceRole.Name", namespaceRoleBinding.Spec.RoleRef.Name)
		return ctrl.Result{}, err
	}

	var processedClusterRoleBindings []kobsiov1alpha1.NamespaceRoleStatusRoleBinding
	var processedRoleBindings []kobsiov1alpha1.NamespaceRoleStatusRoleBinding

	for _, clusterRole := range namespaceRole.Status.ClusterRoles {
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceRoleBinding.Name,
				Labels: map[string]string{
					selectorLabelKeyNRB: namespaceRole.Name,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRole.Name,
			},
			Subjects: namespaceRoleBinding.Spec.Subjects,
		}

		err = ctrl.SetControllerReference(namespaceRole, clusterRoleBinding, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		err = r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name}, existingClusterRoleBinding)
		if err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, clusterRoleBinding); err != nil {
				log.Error(err, "Failed to create ClusterRoleBinding", "ClusterRoleBinding.Name", clusterRoleBinding.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get ClusterRoleBinding", "ClusterRoleBinding.Name", clusterRoleBinding.Name)
			return ctrl.Result{}, err
		} else {
			if err := r.Update(ctx, clusterRoleBinding); err != nil {
				log.Error(err, "Failed to update ClusterRoleBinding", "ClusterRoleBinding.Name", clusterRoleBinding.Name)
				return ctrl.Result{}, err
			}
		}

		processedClusterRoleBindings = append(processedClusterRoleBindings, kobsiov1alpha1.NamespaceRoleStatusRoleBinding{
			Name:      clusterRoleBinding.Name,
			Namespace: clusterRoleBinding.Namespace,
		})
	}

	for _, role := range namespaceRole.Status.Roles {
		roleBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespaceRoleBinding.Name,
				Namespace: role.Namespace,
				Labels: map[string]string{
					selectorLabelKeyNRB: namespaceRole.Name,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			},
			Subjects: namespaceRoleBinding.Spec.Subjects,
		}

		err = ctrl.SetControllerReference(namespaceRole, roleBinding, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		existingRoleBinding := &rbacv1.RoleBinding{}
		err = r.Get(ctx, types.NamespacedName{Namespace: role.Namespace, Name: roleBinding.Name}, existingRoleBinding)
		if err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, roleBinding); err != nil {
				log.Error(err, "Failed to create RoleBinding", "RoleBinding.Name", roleBinding.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get RoleBinding", "RoleBinding.Name", roleBinding.Name)
			return ctrl.Result{}, err
		} else {
			if err := r.Update(ctx, roleBinding); err != nil {
				log.Error(err, "Failed to update RoleBinding", "RoleBinding.Name", roleBinding.Name)
				return ctrl.Result{}, err
			}
		}

		processedRoleBindings = append(processedRoleBindings, kobsiov1alpha1.NamespaceRoleStatusRoleBinding{
			Name:      roleBinding.Name,
			Namespace: roleBinding.Namespace,
		})
	}

	// Get a list of all existing ClusterRoleBindings and RoleBindings, which were
	// created by the operator for the NamespaceRoleBinding.
	existingClusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
	if err := r.List(ctx, existingClusterRoleBindings, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			selectorLabelKeyNR: namespaceRole.Name,
		}),
	}); err != nil {
		log.Error(err, "Failed to list ClusterRoleBindings")
		return ctrl.Result{}, err
	}

	existingRoleBindings := &rbacv1.RoleBindingList{}
	if err := r.List(ctx, existingRoleBindings, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			selectorLabelKeyNR: namespaceRole.Name,
		}),
	}); err != nil {
		log.Error(err, "Failed to list RoleBindings")
		return ctrl.Result{}, err
	}

	// Compare the list of existing ClusterRoleBindings and RoleBindings with the
	// list of processed ClusterRoleBindings and RoleBindings. If a
	// ClusterRoleBinding or RoleBinding exists, which was not processed, we
	// delete it.
	for _, existingClusterRoleBinding := range existingClusterRoleBindings.Items {
		if !wasProcessedNRB(existingClusterRoleBinding.Namespace, existingClusterRoleBinding.Name, processedClusterRoleBindings) {
			if err := r.Delete(ctx, &existingClusterRoleBinding); err != nil {
				log.Error(err, "Failed to delete ClusterRoleBinding", "ClusterRole.Namespace", existingClusterRoleBinding.Namespace, "ClusterRole.Name", existingClusterRoleBinding.Name)
				return ctrl.Result{}, err
			}
		}
	}

	for _, existingRoleBinding := range existingRoleBindings.Items {
		if !wasProcessedNRB(existingRoleBinding.Namespace, existingRoleBinding.Name, processedRoleBindings) {
			if err := r.Delete(ctx, &existingRoleBinding); err != nil {
				log.Error(err, "Failed to delete RoleBindingBinding", "RoleBinding.Namespace", existingRoleBinding.Namespace, "RoleBinding.Name", existingRoleBinding.Name)
				return ctrl.Result{}, err
			}
		}
	}

	namespaceRoleBinding.Status.Selector = fmt.Sprintf("%s=%s", selectorLabelKeyNRB, namespaceRoleBinding.Name)
	namespaceRoleBinding.Status.ClusterRoleBindings = processedClusterRoleBindings
	namespaceRoleBinding.Status.RoleBindings = processedRoleBindings

	err = r.Status().Update(ctx, namespaceRoleBinding)
	if err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func wasProcessedNRB(namespace, name string, processedRoles []kobsiov1alpha1.NamespaceRoleStatusRoleBinding) bool {
	for _, role := range processedRoles {
		if role.Namespace == namespace && role.Name == name {
			return true
		}
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kobsiov1alpha1.NamespaceRoleBinding{}).
		Watches(&kobsiov1alpha1.NamespaceRole{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
