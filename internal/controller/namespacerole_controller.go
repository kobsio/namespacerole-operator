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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	selectorLabelKeyNR = "kobs.io/namespacerole"
)

// NamespaceRoleReconciler reconciles a NamespaceRole object
type NamespaceRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kobs.io,resources=namespaceroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kobs.io,resources=namespaceroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kobs.io,resources=namespaceroles/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;roles,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state. For more
// details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *NamespaceRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconcile NamespaceRole")

	namespaceRole := &kobsiov1alpha1.NamespaceRole{}
	err := r.Get(ctx, req.NamespacedName, namespaceRole)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after
			// reconcile request. Owned objects are automatically garbage
			// collected. For additional cleanup logic use finalizers. Return
			// and don't requeue
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get NamespaceRole")
		return ctrl.Result{}, err
	}

	var processedClusterRoles []kobsiov1alpha1.NamespaceRoleStatusRole
	var processedRoles []kobsiov1alpha1.NamespaceRoleStatusRole

	// If the list of namespaces is empty, we don't need to create any
	// ClusterRoles or Roles, so we can return early.
	if len(namespaceRole.Spec.Namespaces) == 0 {
		log.Info("No namespaces defined")
		return ctrl.Result{}, nil
	}

	// If the NamespaceRole only contains one namespace, which is equal to "*",
	// we create a ClusterRole instead of a Role.
	if len(namespaceRole.Spec.Namespaces) == 1 && namespaceRole.Spec.Namespaces[0] == "*" {
		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceRole.Name,
				Labels: map[string]string{
					selectorLabelKeyNR: namespaceRole.Name,
				},
			},
			Rules: namespaceRole.Spec.Rules,
		}

		err = ctrl.SetControllerReference(namespaceRole, clusterRole, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		existingClusterRole := &rbacv1.ClusterRole{}
		err = r.Get(ctx, types.NamespacedName{Name: clusterRole.Name}, existingClusterRole)
		if err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, clusterRole); err != nil {
				log.Error(err, "Failed to create ClusterRole", "ClusterRole.Name", clusterRole.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get ClusterRole", "ClusterRole.Name", clusterRole.Name)
			return ctrl.Result{}, err
		} else {
			if err := r.Update(ctx, clusterRole); err != nil {
				log.Error(err, "Failed to update ClusterRole", "ClusterRole.Name", clusterRole.Name)
				return ctrl.Result{}, err
			}
		}

		processedClusterRoles = append(processedClusterRoles, kobsiov1alpha1.NamespaceRoleStatusRole{
			Name:      clusterRole.Name,
			Namespace: clusterRole.Namespace,
		})
	} else {
		// Loop through the list of namespaces and create a Role in each
		// namespace.
		for _, namespace := range namespaceRole.Spec.Namespaces {
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespaceRole.Name,
					Namespace: namespace,
					Labels: map[string]string{
						selectorLabelKeyNR: namespaceRole.Name,
					},
				},
				Rules: namespaceRole.Spec.Rules,
			}

			err = ctrl.SetControllerReference(namespaceRole, role, r.Scheme)
			if err != nil {
				return ctrl.Result{}, err
			}

			existingRole := &rbacv1.Role{}
			err = r.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, existingRole)
			if err != nil && errors.IsNotFound(err) {
				if err := r.Create(ctx, role); err != nil {
					log.Error(err, "Failed to create Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
					return ctrl.Result{}, err
				}
			} else if err != nil {
				log.Error(err, "Failed to get Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
				return ctrl.Result{}, err
			} else {
				if err := r.Update(ctx, role); err != nil {
					log.Error(err, "Failed to update Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
					return ctrl.Result{}, err
				}
			}

			processedRoles = append(processedRoles, kobsiov1alpha1.NamespaceRoleStatusRole{
				Name:      role.Name,
				Namespace: role.Namespace,
			})
		}
	}

	// Get a list of all existing ClusterRoles and Roles, which were created by
	// the operator for the NamespaceRole.
	existingClusterRoles := &rbacv1.ClusterRoleList{}
	if err := r.List(ctx, existingClusterRoles, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			selectorLabelKeyNR: namespaceRole.Name,
		}),
	}); err != nil {
		log.Error(err, "Failed to list ClusterRoles")
		return ctrl.Result{}, err
	}

	existingRoles := &rbacv1.RoleList{}
	if err := r.List(ctx, existingRoles, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			selectorLabelKeyNR: namespaceRole.Name,
		}),
	}); err != nil {
		log.Error(err, "Failed to list Roles")
		return ctrl.Result{}, err
	}

	// Compare the list of existing ClusterRoles and Roles with the list of
	// processed ClusterRoles and Roles. If a ClusterRole or Role exists, which
	// was not processed, we delete it.
	for _, existingClusterRole := range existingClusterRoles.Items {
		if !wasProcessedNR(existingClusterRole.Namespace, existingClusterRole.Name, processedClusterRoles) {
			if err := r.Delete(ctx, &existingClusterRole); err != nil {
				log.Error(err, "Failed to delete ClusterRole", "ClusterRole.Namespace", existingClusterRole.Namespace, "ClusterRole.Name", existingClusterRole.Name)
				return ctrl.Result{}, err
			}
		}
	}

	for _, existingRole := range existingRoles.Items {
		if !wasProcessedNR(existingRole.Namespace, existingRole.Name, processedRoles) {
			if err := r.Delete(ctx, &existingRole); err != nil {
				log.Error(err, "Failed to delete Role", "Role.Namespace", existingRole.Namespace, "Role.Name", existingRole.Name)
				return ctrl.Result{}, err
			}
		}
	}

	namespaceRole.Status.Selector = fmt.Sprintf("%s=%s", selectorLabelKeyNR, namespaceRole.Name)
	namespaceRole.Status.ClusterRoles = processedClusterRoles
	namespaceRole.Status.Roles = processedRoles

	err = r.Status().Update(ctx, namespaceRole)
	if err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func wasProcessedNR(namespace, name string, processedRoles []kobsiov1alpha1.NamespaceRoleStatusRole) bool {
	for _, role := range processedRoles {
		if role.Namespace == namespace && role.Name == name {
			return true
		}
	}

	return false
}

// SetupWithManager sets up the controller with the Manager. In the event filter
// we ignore updates to CR status in which case metadata.Generation does not
// change.
func (r *NamespaceRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kobsiov1alpha1.NamespaceRole{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
		}).
		Complete(r)
}
