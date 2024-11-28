package controller

import (
	"context"
	"path/filepath"
	"testing"

	kobsiov1alpha1 "github.com/kobsio/namespacerole-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("Bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config")},
		ErrorIfCRDPathMissing: true,
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = kobsiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("Tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("ClusterRole and ClusterRoleBinding", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		namespaceRole := &kobsiov1alpha1.NamespaceRole{}
		namespaceRoleBinding := &kobsiov1alpha1.NamespaceRoleBinding{}

		BeforeEach(func() {
			By("Create NamespaceRole")
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, namespaceRole)
			if err != nil && errors.IsNotFound(err) {
				resource := &kobsiov1alpha1.NamespaceRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kobs-mygroup1",
					},
					Spec: kobsiov1alpha1.NamespaceRoleSpec{
						Namespaces: []string{"*"},
						Rules: []rbacv1.PolicyRule{{
							APIGroups: []string{""},
							Resources: []string{"namespaces"},
							Verbs:     []string{"get", "list"},
						}},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("Create NamespaceRoleBinding")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, namespaceRoleBinding)
			if err != nil && errors.IsNotFound(err) {
				resource := &kobsiov1alpha1.NamespaceRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kobs-mygroup1",
					},
					Spec: kobsiov1alpha1.NamespaceRoleBindingSpec{
						RoleRef: kobsiov1alpha1.NamespaceRoleBindingSpecRoleRef{
							Name: "kobs-mygroup1",
						},
						Subjects: []rbacv1.Subject{{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "group:default/mygroup1",
						}},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup NamespaceRoleBinding")
			namespaceRoleBinding := &kobsiov1alpha1.NamespaceRoleBinding{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, namespaceRoleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, namespaceRoleBinding)).To(Succeed())

			By("Cleanup NamespaceRole")
			namespaceRole := &kobsiov1alpha1.NamespaceRole{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, namespaceRole)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, namespaceRole)).To(Succeed())
		})

		It("Should successfully reconcile the NamespaceRole and NamespaceRoleBinding", func() {
			By("Reconciling NamespaceRole")
			controllerNamespaceRoleReconciler := &NamespaceRoleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerNamespaceRoleReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kobs-mygroup1"}})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling NamespaceRoleBinding")
			controllerNamespaceRoleBindingReconciler := &NamespaceRoleBindingReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerNamespaceRoleBindingReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kobs-mygroup1"}})
			Expect(err).NotTo(HaveOccurred())

			By("Check ClusterRole")
			role := &rbacv1.ClusterRole{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Name).To(Equal("kobs-mygroup1"))
			Expect(role.Labels).To(Equal(map[string]string{"kobs.io/namespacerole": "kobs-mygroup1"}))
			Expect(role.Rules).To(Equal([]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list"},
			}}))

			By("Check ClusterRoleBinding")
			roleBinding := &rbacv1.ClusterRoleBinding{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup1"}, roleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(roleBinding.Name).To(Equal("kobs-mygroup1"))
			Expect(roleBinding.Labels).To(Equal(map[string]string{"kobs.io/namespacerolebinding": "kobs-mygroup1"}))
			Expect(roleBinding.RoleRef).To(Equal(rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "kobs-mygroup1",
			}))
			Expect(roleBinding.Subjects).To(Equal([]rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "group:default/mygroup1",
			}}))
		})
	})
})

var _ = Describe("Role and RoleBinding", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		namespaceRole := &kobsiov1alpha1.NamespaceRole{}
		namespaceRoleBinding := &kobsiov1alpha1.NamespaceRoleBinding{}

		BeforeEach(func() {
			By("Create NamespaceRole")
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2"}, namespaceRole)
			if err != nil && errors.IsNotFound(err) {
				resource := &kobsiov1alpha1.NamespaceRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kobs-mygroup2",
					},
					Spec: kobsiov1alpha1.NamespaceRoleSpec{
						Namespaces: []string{"default"},
						Rules: []rbacv1.PolicyRule{{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "list"},
						}},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("Create NamespaceRoleBinding")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2"}, namespaceRoleBinding)
			if err != nil && errors.IsNotFound(err) {
				resource := &kobsiov1alpha1.NamespaceRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kobs-mygroup2",
					},
					Spec: kobsiov1alpha1.NamespaceRoleBindingSpec{
						RoleRef: kobsiov1alpha1.NamespaceRoleBindingSpecRoleRef{
							Name: "kobs-mygroup2",
						},
						Subjects: []rbacv1.Subject{{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "group:default/mygroup2",
						}},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup NamespaceRoleBinding")
			namespaceRoleBinding := &kobsiov1alpha1.NamespaceRoleBinding{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2"}, namespaceRoleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, namespaceRoleBinding)).To(Succeed())

			By("Cleanup NamespaceRole")
			namespaceRole := &kobsiov1alpha1.NamespaceRole{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2"}, namespaceRole)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, namespaceRole)).To(Succeed())
		})

		It("Should successfully reconcile the NamespaceRole and NamespaceRoleBinding", func() {
			By("Reconciling NamespaceRole")
			controllerNamespaceRoleReconciler := &NamespaceRoleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerNamespaceRoleReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kobs-mygroup2"}})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling NamespaceRoleBinding")
			controllerNamespaceRoleBindingReconciler := &NamespaceRoleBindingReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err = controllerNamespaceRoleBindingReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "kobs-mygroup2"}})
			Expect(err).NotTo(HaveOccurred())

			By("Check Role")
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2", Namespace: "default"}, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Name).To(Equal("kobs-mygroup2"))
			Expect(role.Namespace).To(Equal("default"))
			Expect(role.Labels).To(Equal(map[string]string{"kobs.io/namespacerole": "kobs-mygroup2"}))
			Expect(role.Rules).To(Equal([]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			}}))

			By("Check RoleBinding")
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "kobs-mygroup2", Namespace: "default"}, roleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(roleBinding.Name).To(Equal("kobs-mygroup2"))
			Expect(roleBinding.Namespace).To(Equal("default"))
			Expect(roleBinding.Labels).To(Equal(map[string]string{"kobs.io/namespacerolebinding": "kobs-mygroup2"}))
			Expect(roleBinding.RoleRef).To(Equal(rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "kobs-mygroup2",
			}))
			Expect(roleBinding.Subjects).To(Equal([]rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "group:default/mygroup2",
			}}))
		})
	})
})
