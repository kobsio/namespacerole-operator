# NamespaceRole Operator

The NamespaceRole Operator is a Kubernetes operator that manages the role-based
access control (RBAC) for namespaces. It allows you to define a set of roles and
role bindings that should be applied to a list of namespaces in a Kubernetes
cluster.

It is intended to simplify the access management for user, which should have
access to a Kuebrnetes cluster via [kobs](https://kobs.io).

For example, if you want that all members of a group `group:default/mygroup1`
have the permissions to list all namespaces and all members of a group
`group:default/mygroup2` can manage all resources in the `monitoring`, `logging`
and `tracing` namespace, you can create a `NamespaceRole` and
`NamespaceRoleBinding` like shown in the following:

```yaml
---
apiVersion: kobs.io/v1alpha1
kind: NamespaceRole
metadata:
  name: kobs-mygroup1
spec:
  namespaces:
    - "*"
  rules:
    - apiGroups:
        - ""
      resources:
        - namespaces
      verbs:
        - get
        - list

---
apiVersion: kobs.io/v1alpha1
kind: NamespaceRoleBinding
metadata:
  name: kobs-mygroup1
spec:
  roleRef:
    name: kobs-mygroup1
  subjects:
    - apiGroup: rbac.authorization.k8s.io
      kind: Group
      name: group:default/mygroup1

---
apiVersion: kobs.io/v1alpha1
kind: NamespaceRole
metadata:
  name: kobs-mygroup2
spec:
  namespaces:
    - monitoring
    - logging
    - tracing
  rules:
    - apiGroups:
        - "*"
      resources:
        - "*"
      verbs:
        - "*"

---
apiVersion: kobs.io/v1alpha1
kind: NamespaceRoleBinding
metadata:
  name: kobs-mygroup2
spec:
  roleRef:
    name: kobs-mygroup2
  subjects:
    - apiGroup: rbac.authorization.k8s.io
      kind: Group
      name: group:default/mygroup2
```

The above example will create a `ClusterRole` and `ClusterRoleBinding`
`kobs-mygroup1` for the first `NamespaceRole` and `NamespaceRoleBinding`. It
will also create three `Role`s and `RoleBinding`s `kobs-mygroup2` for the second
`NamespaceRole` and `NamespaceRoleBinding` in the `monitoring`, `logging` and
`tracing` namespace.

> [!NOTE]
> If the list of namespaces in the `NamespaceRole` only contains one entry with
> the value `*`, a ClusterRole will be created instead of a Role, to grant
> permissions to all namespaces.

## Installation

## Development

After modifying the `*_types.go` files in the `api/v1alpha1` folder always run
the following command to update the generated code for that resource type:

```sh
make generate
```

The above Makefile target will invoke the
[controller-gen](https://sigs.k8s.io/controller-tools) utility to update the
`api/v1alpha1/zz_generated.deepcopy.go` file to ensure our API's Go type
definitons implement the `runtime.Object` interface that all Kind types must
implement.

Once the API is defined with spec/status fields and CRD validation markers, the
CRD manifests can be generated and updated with the following command:

```sh
make manifests
```

This Makefile target will invoke controller-gen to generate the CRD manifests at
`charts/namespacerole-oeprator/crds/kobs.io_*.yaml`.

Deploy the CRD and run the operator locally with the default Kubernetes config
file present at `$HOME/.kube/config`:

```sh
k apply -f charts/namespacerole-oeprator/crds/kobs.io_namespaceroles.yaml
k apply -f charts/namespacerole-oeprator/crds/kobs.io_namespacerolebindings.yaml

make run
```
