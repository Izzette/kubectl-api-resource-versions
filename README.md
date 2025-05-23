# kubectl-api-resource-versions

<img align="left" width="128" height="128" alt="Kubernetes Logo" src="https://raw.githubusercontent.com/cncf/artwork/main/projects/kubernetes/icon/color/kubernetes-icon-color.png">

A [`kubectl` plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that provides a comprehensive view of all available API resources and their versions in a Kubernetes cluster.

[![Go Version](https://img.shields.io/badge/go-1.24-blue)]() [![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/Izzette/kubectl-api-resource-versions)](https://goreportcard.com/report/github.com/Izzette/kubectl-api-resource-versions)

---

## Features

- List API resources with their available group versions in a single view
- Filter by API group, namespaced status, and preferred API group versions
- Multiple output formats: `wide` (default), `name` (kubectl-compatible)
- Sorting by resource name or kind
- Works with any Kubernetes cluster (v1.20+)
- Supports in-cluster and out-of-cluster configurations

## Why?

The `kubectl api-resources` command lists API resources, but only shows the preferred version for each resource—not all available group versions.

For example, in a cluster where the `gateway.networking.k8s.io` API group has both `v1` and `v1beta1`, you might run:

```shell
kubectl get httproutes.gateway.networking.k8s.io --all-namespaces
```

This lists `HTTPRoute` resources in the preferred version (`v1`), but doesn’t show other versions like `v1beta1`. Also, `ReferenceGrant` only exists in `v1beta1`, so you won’t see it with the `v1` API version. `kubectl api-resources` just tells you `HTTPRoute` is available in `v1` and `ReferenceGrant` in `v1beta1`.

On the other hand, `kubectl api-versions` lists all API versions, but not which resources are in each version. For example, the `resources.teleport.dev` API group might have `v1`, `v2`, `v3`, `v5`, and `v6`, but only `TeleportRole` is in `v5` and `v6`, and `TeleportUser` is only in `v2`.

Neither command gives a complete view of resources by version. To find which API versions a resource supports, you have to combine both commands and check each combination yourself.

Here’s a zsh script that tries to do this for resources with the `list` verb:

```shell
#!/usr/bin/env zsh

setopt SH_WORD_SPLIT

typeset -A api_versions=()
while IFS= read -r api_group_version <&3; do
  if [[ $api_group_version == */* ]]; then
    IFS=/ read -r api_group version <<< "$api_group_version"
  else
    api_group=''
    version="$api_group_version"
  fi
  if [[ -n ${api_versions[$api_group]} ]]; then
    api_versions[$api_group]+=" "
  fi
  api_versions[$api_group]+="$version"
done 3< <(kubectl api-versions)

while IFS=. read -r resource api_group <&3; do
  if [[ -z ${api_versions[$api_group]} ]]; then
    continue
  fi
  for version in "${api_versions[$api_group]}"; do
    versioned_resource="$resource.$version.$api_group"
    if kubectl get "$versioned_resource" > /dev/null 2>&1; then
      echo "$versioned_resource"
    fi
  done
done 3< <(kubectl api-resources --verbs='list' --output='name')
```

This script shows how complicated it is, and still gives errors for missing combinations.

This project aims to solve that by letting you easily list all API versions for each resource. For example, you can run:

```shell
kubectl api-resource-versions --verbs='get,list' --output='name'
```

And get a full list of resources and their supported API versions, ready to use with `kubectl get`.

## Installation

<!--
### Via Krew

```shell
kubectl krew install api-resource-versions
```
-->

### Install with `go install`

With this method, you can install the plugin directly from the source code using Go's package manager.

```shell
go install github.com/Izzette/kubectl-api-resource-versions/cmd/kubectl-api_resource_versions@latest

# Optionally install the completion script to your $GOPATH/bin
curl -fLo "$(go env GOPATH)/bin/kubectl_complete-api_resource_versions" \
  https://raw.githubusercontent.com/Izzette/kubectl-api-resource-versions/refs/heads/main/kubectl_complete-api_resource_versions
chmod +x "$(go env GOPATH)/bin/kubectl_complete-api_resource_versions"
```

### Manual Installation

1. Clone the repository:
   ```shell
   git clone https://github.com/Izzette/kubectl-api-resource-versions.git
   cd kubectl-api-resource-versions
   ```

2. Build the plugin:
   ```shell
   make build
   ```

3. Move the binary to your PATH (e.g., `/usr/local/bin`):
   ```shell
   sudo cp kubectl-api_resource_versions kubectl_complete-api_resource_versions /usr/local/bin/
   ```

### Standalone usage

You can also use the plugin without installing it globally. Just run the binary directly:

```shell
make build
./kubectl-api_resource_versions
```

## Usage

### Basic Usage

List all API resources with versions (including deprecated/unstable):

```shell
kubectl api-resource-versions
```

### More examples

Filter to non-preferred versions (these may be unstable APIs or deprecated):
```shell
kubectl api-resource-versions --preferred='false'
```

Filter to resources in specific API group:
```shell
kubectl api-resource-versions --api-group='apps'
```

List non-namespaced resources:
```shell
kubectl api-resource-versions --namespaced='false'
```

Show output in kubectl `name` format, and list those resources:
```shell
kubectl api-resource-versions --api-group='apps' --verbs='list,get' --namespaced='false' --output='name' |
  xargs -n1 kubectl get --show-kind
```

### Output

The tabular output format is similar to `kubectl api-resources`, but with an additional column for which API version is preferred for each resource.

<details>
<summary>Example tabular output format</summary>

```console
$ kubectl api-resource-versions --api-group=''
NAME                     SHORTNAMES   APIVERSION   NAMESPACED   KIND                    PREFERRED
bindings                              v1           true         Binding                 true
componentstatuses        cs           v1           false        ComponentStatus         true
configmaps               cm           v1           true         ConfigMap               true
endpoints                ep           v1           true         Endpoints               true
events                   ev           v1           true         Event                   true
limitranges              limits       v1           true         LimitRange              true
namespaces               ns           v1           false        Namespace               true
nodes                    no           v1           false        Node                    true
persistentvolumeclaims   pvc          v1           true         PersistentVolumeClaim   true
persistentvolumes        pv           v1           false        PersistentVolume        true
pods                     po           v1           true         Pod                     true
podtemplates                          v1           true         PodTemplate             true
replicationcontrollers   rc           v1           true         ReplicationController   true
resourcequotas           quota        v1           true         ResourceQuota           true
secrets                               v1           true         Secret                  true
serviceaccounts          sa           v1           true         ServiceAccount          true
services                 svc          v1           true         Service                 true
$ kubectl api-resource-versions --api-group='admissionregistration.k8s.io'
NAME                                SHORTNAMES   APIVERSION                        NAMESPACED   KIND                               PREFERRED
mutatingwebhookconfigurations                    admissionregistration.k8s.io/v1   false        MutatingWebhookConfiguration       true
validatingadmissionpolicies                      admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicy          true
validatingadmissionpolicybindings                admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicyBinding   true
validatingwebhookconfigurations                  admissionregistration.k8s.io/v1   false        ValidatingWebhookConfiguration     true
$ kubectl api-resource-versions --api-group='apiextensions.k8s.io'
NAME                        SHORTNAMES   APIVERSION                NAMESPACED   KIND                       PREFERRED
customresourcedefinitions   crd,crds     apiextensions.k8s.io/v1   false        CustomResourceDefinition   true
$ kubectl api-resource-versions --api-group='apiregistration.k8s.io'
NAME          SHORTNAMES   APIVERSION                  NAMESPACED   KIND         PREFERRED
apiservices                apiregistration.k8s.io/v1   false        APIService   true
$ kubectl api-resource-versions --api-group='apps'
NAME                  SHORTNAMES   APIVERSION   NAMESPACED   KIND                 PREFERRED
controllerrevisions                apps/v1      true         ControllerRevision   true
daemonsets            ds           apps/v1      true         DaemonSet            true
deployments           deploy       apps/v1      true         Deployment           true
replicasets           rs           apps/v1      true         ReplicaSet           true
statefulsets          sts          apps/v1      true         StatefulSet          true
$ kubectl api-resource-versions --api-group='authentication.k8s.io'
NAME                 SHORTNAMES   APIVERSION                 NAMESPACED   KIND                PREFERRED
selfsubjectreviews                authentication.k8s.io/v1   false        SelfSubjectReview   true
tokenreviews                      authentication.k8s.io/v1   false        TokenReview         true
$ kubectl api-resource-versions --api-group='authorization.k8s.io'
NAME                        SHORTNAMES   APIVERSION                NAMESPACED   KIND                       PREFERRED
localsubjectaccessreviews                authorization.k8s.io/v1   true         LocalSubjectAccessReview   true
selfsubjectaccessreviews                 authorization.k8s.io/v1   false        SelfSubjectAccessReview    true
selfsubjectrulesreviews                  authorization.k8s.io/v1   false        SelfSubjectRulesReview     true
subjectaccessreviews                     authorization.k8s.io/v1   false        SubjectAccessReview        true
$ kubectl api-resource-versions --api-group='autoscaling'
NAME                       SHORTNAMES   APIVERSION       NAMESPACED   KIND                      PREFERRED
horizontalpodautoscalers   hpa          autoscaling/v2   true         HorizontalPodAutoscaler   true
horizontalpodautoscalers   hpa          autoscaling/v1   true         HorizontalPodAutoscaler   false
$ kubectl api-resource-versions --api-group='autoscaling.x-k8s.io'
NAME                   SHORTNAMES         APIVERSION                     NAMESPACED   KIND                  PREFERRED
provisioningrequests   provreq,provreqs   autoscaling.x-k8s.io/v1        true         ProvisioningRequest   true
provisioningrequests   provreq,provreqs   autoscaling.x-k8s.io/v1beta1   true         ProvisioningRequest   false
$ kubectl api-resource-versions --api-group='batch'
NAME       SHORTNAMES   APIVERSION   NAMESPACED   KIND      PREFERRED
cronjobs   cj           batch/v1     true         CronJob   true
jobs                    batch/v1     true         Job       true
$ kubectl api-resource-versions --api-group='certificates.k8s.io'
NAME                         SHORTNAMES   APIVERSION               NAMESPACED   KIND                        PREFERRED
certificatesigningrequests   csr          certificates.k8s.io/v1   false        CertificateSigningRequest   true
$ kubectl api-resource-versions --api-group='coordination.k8s.io'
NAME     SHORTNAMES   APIVERSION               NAMESPACED   KIND    PREFERRED
leases                coordination.k8s.io/v1   true         Lease   true
$ kubectl api-resource-versions --api-group='discovery.k8s.io'
NAME             SHORTNAMES   APIVERSION            NAMESPACED   KIND            PREFERRED
endpointslices                discovery.k8s.io/v1   true         EndpointSlice   true
$ kubectl api-resource-versions --api-group='events.k8s.io'
NAME     SHORTNAMES   APIVERSION         NAMESPACED   KIND    PREFERRED
events   ev           events.k8s.io/v1   true         Event   true
$ kubectl api-resource-versions --api-group='flowcontrol.apiserver.k8s.io'
NAME                          SHORTNAMES   APIVERSION                        NAMESPACED   KIND                         PREFERRED
flowschemas                                flowcontrol.apiserver.k8s.io/v1   false        FlowSchema                   true
prioritylevelconfigurations                flowcontrol.apiserver.k8s.io/v1   false        PriorityLevelConfiguration   true
$ kubectl api-resource-versions --api-group='gateway.networking.k8s.io'
NAME              SHORTNAMES   APIVERSION                          NAMESPACED   KIND             PREFERRED
gatewayclasses    gc           gateway.networking.k8s.io/v1        false        GatewayClass     true
gatewayclasses    gc           gateway.networking.k8s.io/v1beta1   false        GatewayClass     false
gateways          gtw          gateway.networking.k8s.io/v1        true         Gateway          true
gateways          gtw          gateway.networking.k8s.io/v1beta1   true         Gateway          false
httproutes                     gateway.networking.k8s.io/v1        true         HTTPRoute        true
httproutes                     gateway.networking.k8s.io/v1beta1   true         HTTPRoute        false
referencegrants   refgrant     gateway.networking.k8s.io/v1beta1   true         ReferenceGrant   false
$ kubectl api-resource-versions --api-group='metrics.k8s.io'
NAME    SHORTNAMES   APIVERSION               NAMESPACED   KIND          PREFERRED
nodes                metrics.k8s.io/v1beta1   false        NodeMetrics   true
pods                 metrics.k8s.io/v1beta1   true         PodMetrics    true
$ kubectl api-resource-versions --api-group='networking.k8s.io'
NAME              SHORTNAMES   APIVERSION             NAMESPACED   KIND            PREFERRED
ingressclasses                 networking.k8s.io/v1   false        IngressClass    true
ingresses         ing          networking.k8s.io/v1   true         Ingress         true
networkpolicies   netpol       networking.k8s.io/v1   true         NetworkPolicy   true
$ kubectl api-resource-versions --api-group='node.k8s.io'
NAME             SHORTNAMES   APIVERSION       NAMESPACED   KIND           PREFERRED
runtimeclasses                node.k8s.io/v1   false        RuntimeClass   true
$ kubectl api-resource-versions --api-group='policy'
NAME                   SHORTNAMES   APIVERSION   NAMESPACED   KIND                  PREFERRED
poddisruptionbudgets   pdb          policy/v1    true         PodDisruptionBudget   true
$ kubectl api-resource-versions --api-group='rbac.authorization.k8s.io'
NAME                  SHORTNAMES   APIVERSION                     NAMESPACED   KIND                 PREFERRED
clusterrolebindings                rbac.authorization.k8s.io/v1   false        ClusterRoleBinding   true
clusterroles                       rbac.authorization.k8s.io/v1   false        ClusterRole          true
rolebindings                       rbac.authorization.k8s.io/v1   true         RoleBinding          true
roles                              rbac.authorization.k8s.io/v1   true         Role                 true
$ kubectl api-resource-versions --api-group='scheduling.k8s.io'
NAME              SHORTNAMES   APIVERSION             NAMESPACED   KIND            PREFERRED
priorityclasses   pc           scheduling.k8s.io/v1   false        PriorityClass   true
$ kubectl api-resource-versions --api-group='snapshot.storage.k8s.io'
NAME                     SHORTNAMES          APIVERSION                        NAMESPACED   KIND                    PREFERRED
volumesnapshotclasses    vsclass,vsclasses   snapshot.storage.k8s.io/v1        false        VolumeSnapshotClass     true
volumesnapshotclasses    vsclass,vsclasses   snapshot.storage.k8s.io/v1beta1   false        VolumeSnapshotClass     false
volumesnapshotcontents   vsc,vscs            snapshot.storage.k8s.io/v1        false        VolumeSnapshotContent   true
volumesnapshotcontents   vsc,vscs            snapshot.storage.k8s.io/v1beta1   false        VolumeSnapshotContent   false
volumesnapshots          vs                  snapshot.storage.k8s.io/v1        true         VolumeSnapshot          true
volumesnapshots          vs                  snapshot.storage.k8s.io/v1beta1   true         VolumeSnapshot          false
$ kubectl api-resource-versions --api-group='storage.k8s.io'
NAME                   SHORTNAMES   APIVERSION          NAMESPACED   KIND                 PREFERRED
csidrivers                          storage.k8s.io/v1   false        CSIDriver            true
csinodes                            storage.k8s.io/v1   false        CSINode              true
csistoragecapacities                storage.k8s.io/v1   true         CSIStorageCapacity   true
storageclasses         sc           storage.k8s.io/v1   false        StorageClass         true
volumeattachments                   storage.k8s.io/v1   false        VolumeAttachment     true
```
</details>

<details>
<summary>Example tabular output format (wide)</summary>

```console
$ kubectl api-resource-versions --api-group='' --output='wide'
NAME                     SHORTNAMES   APIVERSION   NAMESPACED   KIND                    PREFERRED   VERBS                                                        CATEGORIES
bindings                              v1           true         Binding                 true        create
componentstatuses        cs           v1           false        ComponentStatus         true        get,list
configmaps               cm           v1           true         ConfigMap               true        create,delete,deletecollection,get,list,patch,update,watch
endpoints                ep           v1           true         Endpoints               true        create,delete,deletecollection,get,list,patch,update,watch
events                   ev           v1           true         Event                   true        create,delete,deletecollection,get,list,patch,update,watch
limitranges              limits       v1           true         LimitRange              true        create,delete,deletecollection,get,list,patch,update,watch
namespaces               ns           v1           false        Namespace               true        create,delete,get,list,patch,update,watch
nodes                    no           v1           false        Node                    true        create,delete,deletecollection,get,list,patch,update,watch
persistentvolumeclaims   pvc          v1           true         PersistentVolumeClaim   true        create,delete,deletecollection,get,list,patch,update,watch
persistentvolumes        pv           v1           false        PersistentVolume        true        create,delete,deletecollection,get,list,patch,update,watch
pods                     po           v1           true         Pod                     true        create,delete,deletecollection,get,list,patch,update,watch   all
podtemplates                          v1           true         PodTemplate             true        create,delete,deletecollection,get,list,patch,update,watch
replicationcontrollers   rc           v1           true         ReplicationController   true        create,delete,deletecollection,get,list,patch,update,watch   all
resourcequotas           quota        v1           true         ResourceQuota           true        create,delete,deletecollection,get,list,patch,update,watch
secrets                               v1           true         Secret                  true        create,delete,deletecollection,get,list,patch,update,watch
serviceaccounts          sa           v1           true         ServiceAccount          true        create,delete,deletecollection,get,list,patch,update,watch
services                 svc          v1           true         Service                 true        create,delete,deletecollection,get,list,patch,update,watch   all
$ kubectl api-resource-versions --api-group='admissionregistration.k8s.io' --output='wide'
NAME                                SHORTNAMES   APIVERSION                        NAMESPACED   KIND                               PREFERRED   VERBS                                                        CATEGORIES
mutatingwebhookconfigurations                    admissionregistration.k8s.io/v1   false        MutatingWebhookConfiguration       true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
validatingadmissionpolicies                      admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicy          true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
validatingadmissionpolicybindings                admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicyBinding   true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
validatingwebhookconfigurations                  admissionregistration.k8s.io/v1   false        ValidatingWebhookConfiguration     true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
$ kubectl api-resource-versions --api-group='apiextensions.k8s.io' --output='wide'
NAME                        SHORTNAMES   APIVERSION                NAMESPACED   KIND                       PREFERRED   VERBS                                                        CATEGORIES
customresourcedefinitions   crd,crds     apiextensions.k8s.io/v1   false        CustomResourceDefinition   true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
$ kubectl api-resource-versions --api-group='apiregistration.k8s.io' --output='wide'
NAME          SHORTNAMES   APIVERSION                  NAMESPACED   KIND         PREFERRED   VERBS                                                        CATEGORIES
apiservices                apiregistration.k8s.io/v1   false        APIService   true        create,delete,deletecollection,get,list,patch,update,watch   api-extensions
$ kubectl api-resource-versions --api-group='apps' --output='wide'
NAME                  SHORTNAMES   APIVERSION   NAMESPACED   KIND                 PREFERRED   VERBS                                                        CATEGORIES
controllerrevisions                apps/v1      true         ControllerRevision   true        create,delete,deletecollection,get,list,patch,update,watch
daemonsets            ds           apps/v1      true         DaemonSet            true        create,delete,deletecollection,get,list,patch,update,watch   all
deployments           deploy       apps/v1      true         Deployment           true        create,delete,deletecollection,get,list,patch,update,watch   all
replicasets           rs           apps/v1      true         ReplicaSet           true        create,delete,deletecollection,get,list,patch,update,watch   all
statefulsets          sts          apps/v1      true         StatefulSet          true        create,delete,deletecollection,get,list,patch,update,watch   all
$ kubectl api-resource-versions --api-group='authentication.k8s.io' --output='wide'
NAME                 SHORTNAMES   APIVERSION                 NAMESPACED   KIND                PREFERRED   VERBS    CATEGORIES
selfsubjectreviews                authentication.k8s.io/v1   false        SelfSubjectReview   true        create
tokenreviews                      authentication.k8s.io/v1   false        TokenReview         true        create
$ kubectl api-resource-versions --api-group='authorization.k8s.io' --output='wide'
NAME                        SHORTNAMES   APIVERSION                NAMESPACED   KIND                       PREFERRED   VERBS    CATEGORIES
localsubjectaccessreviews                authorization.k8s.io/v1   true         LocalSubjectAccessReview   true        create
selfsubjectaccessreviews                 authorization.k8s.io/v1   false        SelfSubjectAccessReview    true        create
selfsubjectrulesreviews                  authorization.k8s.io/v1   false        SelfSubjectRulesReview     true        create
subjectaccessreviews                     authorization.k8s.io/v1   false        SubjectAccessReview        true        create
$ kubectl api-resource-versions --api-group='autoscaling' --output='wide'
NAME                       SHORTNAMES   APIVERSION       NAMESPACED   KIND                      PREFERRED   VERBS                                                        CATEGORIES
horizontalpodautoscalers   hpa          autoscaling/v2   true         HorizontalPodAutoscaler   true        create,delete,deletecollection,get,list,patch,update,watch   all
horizontalpodautoscalers   hpa          autoscaling/v1   true         HorizontalPodAutoscaler   false       create,delete,deletecollection,get,list,patch,update,watch   all
$ kubectl api-resource-versions --api-group='autoscaling.x-k8s.io' --output='wide'
NAME                   SHORTNAMES         APIVERSION                     NAMESPACED   KIND                  PREFERRED   VERBS                                                        CATEGORIES
provisioningrequests   provreq,provreqs   autoscaling.x-k8s.io/v1        true         ProvisioningRequest   true        delete,deletecollection,get,list,patch,create,update,watch
provisioningrequests   provreq,provreqs   autoscaling.x-k8s.io/v1beta1   true         ProvisioningRequest   false       delete,deletecollection,get,list,patch,create,update,watch
$ kubectl api-resource-versions --api-group='batch' --output='wide'
NAME       SHORTNAMES   APIVERSION   NAMESPACED   KIND      PREFERRED   VERBS                                                        CATEGORIES
cronjobs   cj           batch/v1     true         CronJob   true        create,delete,deletecollection,get,list,patch,update,watch   all
jobs                    batch/v1     true         Job       true        create,delete,deletecollection,get,list,patch,update,watch   all
$ kubectl api-resource-versions --api-group='certificates.k8s.io' --output='wide'
NAME                         SHORTNAMES   APIVERSION               NAMESPACED   KIND                        PREFERRED   VERBS                                                        CATEGORIES
certificatesigningrequests   csr          certificates.k8s.io/v1   false        CertificateSigningRequest   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='coordination.k8s.io' --output='wide'
NAME     SHORTNAMES   APIVERSION               NAMESPACED   KIND    PREFERRED   VERBS                                                        CATEGORIES
leases                coordination.k8s.io/v1   true         Lease   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='discovery.k8s.io' --output='wide'
NAME             SHORTNAMES   APIVERSION            NAMESPACED   KIND            PREFERRED   VERBS                                                        CATEGORIES
endpointslices                discovery.k8s.io/v1   true         EndpointSlice   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='events.k8s.io' --output='wide'
NAME     SHORTNAMES   APIVERSION         NAMESPACED   KIND    PREFERRED   VERBS                                                        CATEGORIES
events   ev           events.k8s.io/v1   true         Event   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='flowcontrol.apiserver.k8s.io' --output='wide'
NAME                          SHORTNAMES   APIVERSION                        NAMESPACED   KIND                         PREFERRED   VERBS                                                        CATEGORIES
flowschemas                                flowcontrol.apiserver.k8s.io/v1   false        FlowSchema                   true        create,delete,deletecollection,get,list,patch,update,watch
prioritylevelconfigurations                flowcontrol.apiserver.k8s.io/v1   false        PriorityLevelConfiguration   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='gateway.networking.k8s.io' --output='wide'
NAME              SHORTNAMES   APIVERSION                          NAMESPACED   KIND             PREFERRED   VERBS                                                        CATEGORIES
gatewayclasses    gc           gateway.networking.k8s.io/v1        false        GatewayClass     true        delete,deletecollection,get,list,patch,create,update,watch   gateway-api
gatewayclasses    gc           gateway.networking.k8s.io/v1beta1   false        GatewayClass     false       delete,deletecollection,get,list,patch,create,update,watch   gateway-api
gateways          gtw          gateway.networking.k8s.io/v1        true         Gateway          true        delete,deletecollection,get,list,patch,create,update,watch   gateway-api
gateways          gtw          gateway.networking.k8s.io/v1beta1   true         Gateway          false       delete,deletecollection,get,list,patch,create,update,watch   gateway-api
httproutes                     gateway.networking.k8s.io/v1        true         HTTPRoute        true        delete,deletecollection,get,list,patch,create,update,watch   gateway-api
httproutes                     gateway.networking.k8s.io/v1beta1   true         HTTPRoute        false       delete,deletecollection,get,list,patch,create,update,watch   gateway-api
referencegrants   refgrant     gateway.networking.k8s.io/v1beta1   true         ReferenceGrant   false       delete,deletecollection,get,list,patch,create,update,watch   gateway-api
$ kubectl api-resource-versions --api-group='metrics.k8s.io' --output='wide'
NAME    SHORTNAMES   APIVERSION               NAMESPACED   KIND          PREFERRED   VERBS      CATEGORIES
nodes                metrics.k8s.io/v1beta1   false        NodeMetrics   true        get,list
pods                 metrics.k8s.io/v1beta1   true         PodMetrics    true        get,list
$ kubectl api-resource-versions --api-group='networking.k8s.io' --output='wide'
NAME              SHORTNAMES   APIVERSION             NAMESPACED   KIND            PREFERRED   VERBS                                                        CATEGORIES
ingressclasses                 networking.k8s.io/v1   false        IngressClass    true        create,delete,deletecollection,get,list,patch,update,watch
ingresses         ing          networking.k8s.io/v1   true         Ingress         true        create,delete,deletecollection,get,list,patch,update,watch
networkpolicies   netpol       networking.k8s.io/v1   true         NetworkPolicy   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='node.k8s.io' --output='wide'
NAME             SHORTNAMES   APIVERSION       NAMESPACED   KIND           PREFERRED   VERBS                                                        CATEGORIES
runtimeclasses                node.k8s.io/v1   false        RuntimeClass   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='policy' --output='wide'
NAME                   SHORTNAMES   APIVERSION   NAMESPACED   KIND                  PREFERRED   VERBS                                                        CATEGORIES
poddisruptionbudgets   pdb          policy/v1    true         PodDisruptionBudget   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='rbac.authorization.k8s.io' --output='wide'
NAME                  SHORTNAMES   APIVERSION                     NAMESPACED   KIND                 PREFERRED   VERBS                                                        CATEGORIES
clusterrolebindings                rbac.authorization.k8s.io/v1   false        ClusterRoleBinding   true        create,delete,deletecollection,get,list,patch,update,watch
clusterroles                       rbac.authorization.k8s.io/v1   false        ClusterRole          true        create,delete,deletecollection,get,list,patch,update,watch
rolebindings                       rbac.authorization.k8s.io/v1   true         RoleBinding          true        create,delete,deletecollection,get,list,patch,update,watch
roles                              rbac.authorization.k8s.io/v1   true         Role                 true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='scheduling.k8s.io' --output='wide'
NAME              SHORTNAMES   APIVERSION             NAMESPACED   KIND            PREFERRED   VERBS                                                        CATEGORIES
priorityclasses   pc           scheduling.k8s.io/v1   false        PriorityClass   true        create,delete,deletecollection,get,list,patch,update,watch
$ kubectl api-resource-versions --api-group='snapshot.storage.k8s.io' --output='wide'
NAME                     SHORTNAMES          APIVERSION                        NAMESPACED   KIND                    PREFERRED   VERBS                                                        CATEGORIES
volumesnapshotclasses    vsclass,vsclasses   snapshot.storage.k8s.io/v1        false        VolumeSnapshotClass     true        delete,deletecollection,get,list,patch,create,update,watch
volumesnapshotclasses    vsclass,vsclasses   snapshot.storage.k8s.io/v1beta1   false        VolumeSnapshotClass     false       delete,deletecollection,get,list,patch,create,update,watch
volumesnapshotcontents   vsc,vscs            snapshot.storage.k8s.io/v1        false        VolumeSnapshotContent   true        delete,deletecollection,get,list,patch,create,update,watch
volumesnapshotcontents   vsc,vscs            snapshot.storage.k8s.io/v1beta1   false        VolumeSnapshotContent   false       delete,deletecollection,get,list,patch,create,update,watch
volumesnapshots          vs                  snapshot.storage.k8s.io/v1        true         VolumeSnapshot          true        delete,deletecollection,get,list,patch,create,update,watch
volumesnapshots          vs                  snapshot.storage.k8s.io/v1beta1   true         VolumeSnapshot          false       delete,deletecollection,get,list,patch,create,update,watch
$ kubectl api-resource-versions --api-group='storage.k8s.io' --output='wide'
NAME                   SHORTNAMES   APIVERSION          NAMESPACED   KIND                 PREFERRED   VERBS                                                        CATEGORIES
csidrivers                          storage.k8s.io/v1   false        CSIDriver            true        create,delete,deletecollection,get,list,patch,update,watch
csinodes                            storage.k8s.io/v1   false        CSINode              true        create,delete,deletecollection,get,list,patch,update,watch
csistoragecapacities                storage.k8s.io/v1   true         CSIStorageCapacity   true        create,delete,deletecollection,get,list,patch,update,watch
storageclasses         sc           storage.k8s.io/v1   false        StorageClass         true        create,delete,deletecollection,get,list,patch,update,watch
volumeattachments                   storage.k8s.io/v1   false        VolumeAttachment     true        create,delete,deletecollection,get,list,patch,update,watch
```
</details>

The `name` output format can be used directly with `kubectl` commands to interact with resources in a specific API version.

<details>
<summary>Example name-only output format</summary>

```console
$ kubectl api-resource-versions --api-group='' --output='name'
bindings.v1.
componentstatuses.v1.
configmaps.v1.
endpoints.v1.
events.v1.
limitranges.v1.
namespaces.v1.
nodes.v1.
persistentvolumeclaims.v1.
persistentvolumes.v1.
pods.v1.
podtemplates.v1.
replicationcontrollers.v1.
resourcequotas.v1.
secrets.v1.
serviceaccounts.v1.
services.v1.
$ kubectl api-resource-versions --api-group='admissionregistration.k8s.io' --output='name'
mutatingwebhookconfigurations.v1.admissionregistration.k8s.io
validatingadmissionpolicies.v1.admissionregistration.k8s.io
validatingadmissionpolicybindings.v1.admissionregistration.k8s.io
validatingwebhookconfigurations.v1.admissionregistration.k8s.io
$ kubectl api-resource-versions --api-group='apiextensions.k8s.io' --output='name'
customresourcedefinitions.v1.apiextensions.k8s.io
$ kubectl api-resource-versions --api-group='apiregistration.k8s.io' --output='name'
apiservices.v1.apiregistration.k8s.io
$ kubectl api-resource-versions --api-group='apps' --output='name'
controllerrevisions.v1.apps
daemonsets.v1.apps
deployments.v1.apps
replicasets.v1.apps
statefulsets.v1.apps
$ kubectl api-resource-versions --api-group='authentication.k8s.io' --output='name'
selfsubjectreviews.v1.authentication.k8s.io
tokenreviews.v1.authentication.k8s.io
$ kubectl api-resource-versions --api-group='authorization.k8s.io' --output='name'
localsubjectaccessreviews.v1.authorization.k8s.io
selfsubjectaccessreviews.v1.authorization.k8s.io
selfsubjectrulesreviews.v1.authorization.k8s.io
subjectaccessreviews.v1.authorization.k8s.io
$ kubectl api-resource-versions --api-group='autoscaling' --output='name'
horizontalpodautoscalers.v2.autoscaling
horizontalpodautoscalers.v1.autoscaling
$ kubectl api-resource-versions --api-group='autoscaling.x-k8s.io' --output='name'
provisioningrequests.v1.autoscaling.x-k8s.io
provisioningrequests.v1beta1.autoscaling.x-k8s.io
$ kubectl api-resource-versions --api-group='batch' --output='name'
cronjobs.v1.batch
jobs.v1.batch
$ kubectl api-resource-versions --api-group='certificates.k8s.io' --output='name'
certificatesigningrequests.v1.certificates.k8s.io
$ kubectl api-resource-versions --api-group='coordination.k8s.io' --output='name'
leases.v1.coordination.k8s.io
$ kubectl api-resource-versions --api-group='discovery.k8s.io' --output='name'
endpointslices.v1.discovery.k8s.io
$ kubectl api-resource-versions --api-group='events.k8s.io' --output='name'
events.v1.events.k8s.io
$ kubectl api-resource-versions --api-group='flowcontrol.apiserver.k8s.io' --output='name'
flowschemas.v1.flowcontrol.apiserver.k8s.io
prioritylevelconfigurations.v1.flowcontrol.apiserver.k8s.io
$ kubectl api-resource-versions --api-group='gateway.networking.k8s.io' --output='name'
gatewayclasses.v1.gateway.networking.k8s.io
gatewayclasses.v1beta1.gateway.networking.k8s.io
gateways.v1.gateway.networking.k8s.io
gateways.v1beta1.gateway.networking.k8s.io
httproutes.v1.gateway.networking.k8s.io
httproutes.v1beta1.gateway.networking.k8s.io
referencegrants.v1beta1.gateway.networking.k8s.io
$ kubectl api-resource-versions --api-group='metrics.k8s.io' --output='name'
nodes.v1beta1.metrics.k8s.io
pods.v1beta1.metrics.k8s.io
$ kubectl api-resource-versions --api-group='networking.k8s.io' --output='name'
ingressclasses.v1.networking.k8s.io
ingresses.v1.networking.k8s.io
networkpolicies.v1.networking.k8s.io
$ kubectl api-resource-versions --api-group='node.k8s.io' --output='name'
runtimeclasses.v1.node.k8s.io
$ kubectl api-resource-versions --api-group='policy' --output='name'
poddisruptionbudgets.v1.policy
$ kubectl api-resource-versions --api-group='rbac.authorization.k8s.io' --output='name'
clusterrolebindings.v1.rbac.authorization.k8s.io
clusterroles.v1.rbac.authorization.k8s.io
rolebindings.v1.rbac.authorization.k8s.io
roles.v1.rbac.authorization.k8s.io
$ kubectl api-resource-versions --api-group='scheduling.k8s.io' --output='name'
priorityclasses.v1.scheduling.k8s.io
$ kubectl api-resource-versions --api-group='snapshot.storage.k8s.io' --output='name'
volumesnapshotclasses.v1.snapshot.storage.k8s.io
volumesnapshotclasses.v1beta1.snapshot.storage.k8s.io
volumesnapshotcontents.v1.snapshot.storage.k8s.io
volumesnapshotcontents.v1beta1.snapshot.storage.k8s.io
volumesnapshots.v1.snapshot.storage.k8s.io
volumesnapshots.v1beta1.snapshot.storage.k8s.io
$ kubectl api-resource-versions --api-group='storage.k8s.io' --output='name'
csidrivers.v1.storage.k8s.io
csinodes.v1.storage.k8s.io
csistoragecapacities.v1.storage.k8s.io
storageclasses.v1.storage.k8s.io
volumeattachments.v1.storage.k8s.io
```
</details>

### Command Options

In additional to the normal `kubectl` options, the following options are available:

```text
Flags:
      --api-group string               Limit to resources in the specified API group.
      --cached                         Use the cached list of resources if available.
      --categories strings             Limit to resources that belong to the specified categories.
  -h, --help                           help for api-resource-versions
      --namespaced                     If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. (default true)
      --no-headers                     When using the default or custom-column output format, don't print headers (default print headers).
  -o, --output string                  Output format. One of: (wide, name).
      --preferred                      Filter resources by whether their group version is the preferred one.
      --sort-by string                 If non-empty, sort list of resources using specified field. One of (name, kind).
      --verbs strings                  Limit to resources that support the specified verbs.
```

## Documentation

Full command documentation:

```shell
kubectl api-resource-versions --help
```

Implementation details and API documentation available in the
[project repository](https://github.com/Izzette/kubectl-api-resource-versions) and
[GoDoc](https://pkg.go.dev/github.com/Izzette/kubectl-api-resource-versions).

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Install pre-requisites:
   - Go 1.24.3 or later
   - Python 3.9 or later (for pre-commit)
   - pre-commit (https://pre-commit.com/)
   - Make (GNU Make recommended: https://www.gnu.org/software/make/)
   - Golangci-lint (https://golangci-lint.run/welcome/install/#local-installation):
     - ```shell
       go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
       ```

2. Set up development environment:
   ```shell
   # Install python virtual environment for pre-commit hooks
   pre-commit install
   ```

3. Update documentation accordingly.
   Use Godoc comments for public types and functions: https://go.dev/blog/godoc

4. Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit titles.
   This is required for our automated release process, [Release Please](https://github.com/googleapis/release-please).

5. Open a pull request with a clear description of the changes and why they are needed.
   Include the CHANGELOG entry you would like to see in the release, it doesn't need to be perfect: we can refine it together.

### Development

```console
$ make help
all                            Run all the tests, linters and build the project
build                          Build the project (resulting binary is written to kubectl-api_resource_versions)
buildable                      Check if the project is buildable
clean                          Clean the working directory from binaries, coverage
lint                           Run the linters
```

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgements

⚠️ **Disclaimer**: This project is a derivative work adapted from the Kubernetes [`kubectl`](https://github.com/kubernetes/kubectl) implementation, but is **not** owned, maintained, or endorsed by The Kubernetes Authors. Kubernetes® is a registered trademark of the Linux Foundation, and the author(s) of kubectl-api-resource-versions have no affiliation with the Kubernetes Authors, the CNCF, or Linux Foundation.
