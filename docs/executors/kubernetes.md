---
stage: Verify
group: Runner
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/engineering/ux/technical-writing/#assignments
---

# The Kubernetes executor

GitLab Runner can use Kubernetes to run builds on a Kubernetes cluster. This is
possible with the use of the **Kubernetes** executor.

The **Kubernetes** executor, when used with GitLab CI, connects to the Kubernetes
API in the cluster creating a Pod for each GitLab CI Job. This Pod is made
up of, at the very least, a build container, a helper container, and an additional container for each
`service` defined in the `.gitlab-ci.yml` or `config.toml` files. The names for these containers
are as follows:

- The build container is `build`
- The helper container is `helper`
- The services containers are `svc-X` where `X` is `[0-9]+`

Note that when services and containers are running in the same Kubernetes
pod, they are all sharing the same localhost address. The following restrictions
are then applicable:

- Since GitLab Runner 12.8 and Kubernetes 1.7, the services are accessible via their DNS names. If you are using an older version you will have to use `localhost`.
- You cannot use several services using the same port (e.g., you cannot have two
  `mysql` services at the same time).

## Workflow

The Kubernetes executor divides the build into multiple steps:

1. **Prepare**: Create the Pod against the Kubernetes Cluster.
   This creates the containers required for the build and services to run.
1. **Pre-build**: Clone, restore cache and download artifacts from previous
   stages. This is run on a special container as part of the Pod.
1. **Build**: User build.
1. **Post-build**: Create cache, upload artifacts to GitLab. This also uses
   the special container as part of the Pod.

## Connecting to the Kubernetes API

The following options are provided, which allow you to connect to the Kubernetes API:

- `host`: Optional Kubernetes apiserver host URL (auto-discovery attempted if not specified)
- `cert_file`: Optional Kubernetes apiserver user auth certificate
- `key_file`: Optional Kubernetes apiserver user auth private key
- `ca_file`: Optional Kubernetes apiserver ca certificate

The user account provided must have permission to create, list and attach to Pods in
the specified namespace in order to function.

If you are running GitLab Runner within the Kubernetes cluster you can omit
all of the above fields to have GitLab Runner auto-discover the Kubernetes API. This
is the recommended approach.

If you are running it externally to the Cluster then you will need to set each
of these keywords and make sure that GitLab Runner has access to the Kubernetes API
on the cluster.

## Kubernetes executor interaction diagram

The diagram below depicts the interaction with a GitLab Runner hosted on a Kubernetes cluster and the Kubernetes API. The Kubernetes API is the mechanism that is used by GitLab Runner on Kubernetes to create pods on the cluster. The interaction depicted in this diagram is valid on any Kubernetes cluster, whether that's a turnkey solution hosted on the major public cloud providers or a self-managed Kubernetes installation.

NOTE:
The GitLab Runner [Helm chart](../install/kubernetes.md) has been tested on Google Kubernetes Engine and Azure Container Service.

```mermaid
sequenceDiagram
    participant G as GitLab instance
    participant R as Runner on Kubernetes cluster
    participant Kube as Kubernetes API
    participant P as POD
    R->>+G: Get a CI job.
        loop
        G-->R: ;
    end
    Note over R,G: POST /api/v4/jobs/request
    G->>+R: CI job data.
    R-->>-Kube: Create a POD to run the CI job.
    Note over R,Kube: POST to Kube API
    P->>+P: Execute job
    Note over P: CI build job = Prepare + Pre-build + Build + Post-build
    P->>+G: Job logs
```

## The keywords

The following keywords help to define the behavior of GitLab Runner within Kubernetes.

### CPU requests and limits

| Keyword | Description |
|---------|-------------|
| `cpu_limit` | The CPU allocation given to build containers. |
| `cpu_limit_overwrite_max_allowed` | The max amount the CPU allocation can be written to for build containers. When empty, it disables the cpu limit overwrite feature. |
| `cpu_request` | The CPU allocation requested for build containers. |
| `cpu_request_overwrite_max_allowed` | The max amount the CPU allocation request can be written to for build containers. When empty, it disables the cpu request overwrite feature. |
| `helper_cpu_limit` | The CPU allocation given to build helper containers. |
| `helper_cpu_limit_overwrite_max_allowed` | The max amount the CPU allocation can be written to for helper containers. When empty, it disables the cpu limit overwrite feature. |
| `helper_cpu_request` | The CPU allocation requested for build helper containers. |
| `helper_cpu_request_overwrite_max_allowed` | The max amount the CPU allocation request can be written to for helper containers. When empty, it disables the cpu request overwrite feature. |
| `service_cpu_limit` | The CPU allocation given to build service containers. |
| `service_cpu_limit_overwrite_max_allowed` | The max amount the CPU allocation can be written to for service containers. When empty, it disables the cpu limit overwrite feature. |
| `service_cpu_request` | The CPU allocation requested for build service containers. |
| `service_cpu_request_overwrite_max_allowed` | The max amount the CPU allocation request can be written to for service containers. When empty, it disables the cpu request overwrite feature. |

### Memory requests and limits

| Keyword | Description |
|---------|-------------|
| `memory_limit` | The amount of memory allocated to build containers. |
| `memory_limit_overwrite_max_allowed` | The max amount the memory allocation can be written to for build containers. When empty, it disables the memory limit overwrite feature. |
| `memory_request` | The amount of memory requested from build containers. |
| `memory_request_overwrite_max_allowed` | The max amount the memory allocation request can be written to for build containers. When empty, it disables the memory request overwrite feature. |
| `helper_memory_limit` | The amount of memory allocated to build helper containers. |
| `helper_memory_limit_overwrite_max_allowed` | The max amount the memory allocation can be written to for helper containers. When empty, it disables the memory limit overwrite feature. |
| `helper_memory_request` | The amount of memory requested for build helper containers. |
| `helper_memory_request_overwrite_max_allowed` | The max amount the memory allocation request can be written to for helper containers. When empty, it disables the memory request overwrite feature. |
| `service_memory_limit` | The amount of memory allocated to build service containers. |
| `service_memory_limit_overwrite_max_allowed` | The max amount the memory allocation can be written to for service containers. When empty, it disables the memory limit overwrite feature. |
| `service_memory_request` | The amount of memory requested for build service containers. |
| `service_memory_request_overwrite_max_allowed` | The max amount the memory allocation request can be written to for service containers. When empty, it disables the memory request overwrite feature. |

### Storage requests and limits

| Keyword | Description |
|---------|-------------|
| `ephemeral_storage_limit` | The ephemeral storage limit for build containers. |
| `ephemeral_storage_limit_overwrite_max_allowed` | The max amount the ephemeral storage limit for build containers can be overwritten. When empty, it disables the ephemeral storage limit overwrite feature. |
| `ephemeral_storage_request` | The ephemeral storage request given to build containers. |
| `ephemeral_storage_request_overwrite_max_allowed` | The max amount the ephemeral storage request can be overwritten by for build containers. When empty, it disables the ephemeral storage request overwrite feature. |
| `helper_ephemeral_storage_limit` | The ephemeral storage limit given to helper containers. |
| `helper_ephemeral_storage_limit_overwrite_max_allowed` | The max amount the ephemeral storage limit can be overwritten by for helper containers. When empty, it disables the ephemeral storage request overwrite feature. |
| `helper_ephemeral_storage_request` | The ephemeral storage request given to helper containers. |
| `helper_ephemeral_storage_request_overwrite_max_allowed` | The max amount the ephemeral storage request can be overwritten by for helper containers. When empty, it disables the ephemeral storage request overwrite feature. |
| `service_ephemeral_storage_limit` | The ephemeral storage limit given to service containers. |
| `service_ephemeral_storage_limit_overwrite_max_allowed` | The max amount the ephemeral storage limit can be overwritten by for service containers. When empty, it disables the ephemeral storage request overwrite feature. |
| `service_ephemeral_storage_request` | The ephemeral storage request given to service containers. |
| `service_ephemeral_storage_request_overwrite_max_allowed` | The max amount the ephemeral storage request can be overwritten by for service containers. When empty, it disables the ephemeral storage request overwrite feature. |

### Other keywords

| Keyword | Description |
|---------|-------------|
| `affinity` | Specify affinity rules that determine which node runs the build. Read more about [using affinity](#using-affinity). |
| `allow_privilege_escalation` | Run all containers with the `allowPrivilegeEscalation` flag enabled. When empty, it does not define the `allowPrivilegeEscalation` flag in the container `SecurityContext` and allows Kubernetes to use the default [privilege escalation](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#privilege-escalation) behavior. |
| `bearer_token` | Default bearer token used to launch build pods. |
| `bearer_token_overwrite_allowed` | Boolean to allow projects to specify a bearer token that will be used to create the build pod. |
| `cap_add` | Specify Linux capabilities that should be added to the job pod containers. [Read more about capabilities configuration in Kubernetes executor](#capabilities-configuration). |
| `cap_drop` | Specify Linux capabilities that should be dropped from the job pod containers. [Read more about capabilities configuration in Kubernetes executor](#capabilities-configuration). |
| `helper_image` | (Advanced) [Override the default helper image](../configuration/advanced-configuration.md#helper-image) used to clone repos and upload artifacts. |
| `host_aliases` | List of additional host name aliases that will be added to all containers. [Read more about using extra host aliases](#adding-extra-host-aliases). |
| `image_pull_secrets` | A array of secrets that are used to authenticate Docker image pulling. |
| `namespace` | Namespace in which to run Kubernetes Pods. |
| `namespace_overwrite_allowed` | Regular expression to validate the contents of the namespace overwrite environment variable (documented below). When empty, it disables the namespace overwrite feature. |
| `node_selector` | A `table` of `key=value` pairs in the format of `string=string` (`string:string` in the case of environment variables). Setting this limits the creation of pods to Kubernetes nodes matching all the `key=value` pairs. |
| `node_tolerations` | A `table` of `"key=value" = "Effect"` pairs in the format of `string=string:string`. Setting this allows pods to schedule to nodes with all or a subset of tolerated taints. Only one toleration can be supplied through environment variable configuration. The `key`, `value`, and `effect` match with the corresponding field names in Kubernetes pod toleration configuration. |
| `pod_annotations` | A `table` of `key=value` pairs in the format of `string=string`. This is the list of annotations to be added to each build pod created by the Runner. The value of these can include environment variables for expansion. Pod annotations can be overwritten in each build. |
| `pod_annotations_overwrite_allowed` | Regular expression to validate the contents of the pod annotations overwrite environment variable. When empty, it disables the pod annotations overwrite feature. |
| `pod_labels` | A set of labels to be added to each build pod created by the runner. The value of these can include environment variables for expansion. |
| `pod_security_context` | Configured through the configuration file, this sets a pod security context for the build pod. [Read more about security context](#using-security-context). |
| `poll_interval` | How frequently, in seconds, the runner will poll the Kubernetes pod it has just created to check its status (default = 3). |
| `poll_timeout` | The amount of time, in seconds, that needs to pass before the runner will time out attempting to connect to the container it has just created. Useful for queueing more builds that the cluster can handle at a time (default = 180). |
| `privileged` | Run containers with the privileged flag. |
| `pull_policy` | Specify the image pull policy: `never`, `if-not-present`, `always`. The cluster's image [default pull policy](https://kubernetes.io/docs/concepts/containers/images/#updating-images) will be used if not set. See also [`if-not-present` security considerations](../security/index.md#usage-of-private-docker-images-with-if-not-present-pull-policy) and [using pull policies](#using-pull-policies) for more information and setting multiple policies. |
| `service_account` | Default service account job/executor pods use to talk to Kubernetes API. |
| `service_account_overwrite_allowed` | Regular expression to validate the contents of the service account overwrite environment variable. When empty, it disables the service account overwrite feature. |
| `services` | [Since GitLab Runner 12.5](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/4470), list of [services](https://docs.gitlab.com/ee/ci/services/) attached to the build container using the [sidecar pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar). Read more about [using services](#using-services). |
| `terminationGracePeriodSeconds` | Duration after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. |
| `volumes` | Configured through the configuration file, the list of volumes that will be mounted in the build container. [Read more about using volumes](#using-volumes). |
| `dns_policy` | Specify the DNS policy that should be used when constructing the pod: `none`, `default`, `cluster-first`, `cluster-first-with-host-net`. The Kubernetes default (`cluster-first`) will be used if not set. |
| `dns_config` | Specify the DNS configuration that should be used when constructing the pod. [Read more about using pod's DNS config](#pods-dns-config). |

### Configuring executor Service Account

You can set the `KUBERNETES_SERVICE_ACCOUNT` environment variable or use `--kubernetes-service-account` flag.

### Overwriting Kubernetes Namespace

Additionally, Kubernetes namespace can be overwritten on `.gitlab-ci.yml` file, by using the variable
`KUBERNETES_NAMESPACE_OVERWRITE`.

This approach allows you to create a new isolated namespace dedicated for CI purposes, and deploy a custom
set of Pods. The `Pods` spawned by the runner will take place on the overwritten namespace, for simple
and straight forward access between containers during the CI stages.

``` yaml
variables:
  KUBERNETES_NAMESPACE_OVERWRITE: ci-${CI_COMMIT_REF_SLUG}
```

Furthermore, to ensure only designated namespaces will be used during CI runs, set the configuration
`namespace_overwrite_allowed` with an appropriate regular expression. When left empty the overwrite behavior is
disabled.

### Overwriting Kubernetes Default Service Account

Additionally, the Kubernetes service account can be overwritten in the `.gitlab-ci.yml` file by using the variable
`KUBERNETES_SERVICE_ACCOUNT_OVERWRITE`.

This approach allows you to specify a service account that is attached to the namespace, which is useful when dealing
with complex RBAC configurations.

``` yaml
variables:
  KUBERNETES_SERVICE_ACCOUNT_OVERWRITE: ci-service-account
```

To ensure only designated service accounts will be used during CI runs, set the configuration
`service_account_overwrite_allowed` or set the environment variable `KUBERNETES_SERVICE_ACCOUNT_OVERWRITE_ALLOWED`
with an appropriate regular expression. When left empty the overwrite behavior is disabled.

### Setting Bearer Token to be Used When Making Kubernetes API calls

In conjunction with setting the namespace and service account as mentioned above, you may set the
bearer token used when making API calls to create the build pods. This will allow project owners to
use project secret variables to specify a bearer token. When specifying the bearer token, you must
set the `Host` configuration keyword.

``` yaml
variables:
  KUBERNETES_BEARER_TOKEN: thebearertokenfromanothernamespace
```

### Overwriting pod annotations

Additionally, Kubernetes pod annotations can be overwritten on the `.gitlab-ci.yml` file, by using `KUBERNETES_POD_ANNOTATIONS_*` for variables and `key=value` for the value. The pod annotations will be overwritten to the `key=value`. Multiple annotations can be applied. For example:

```yaml
variables:
  KUBERNETES_POD_ANNOTATIONS_1: "Key1=Val1"
  KUBERNETES_POD_ANNOTATIONS_2: "Key2=Val2"
  KUBERNETES_POD_ANNOTATIONS_3: "Key3=Val3"
```

NOTE:
You must specify [`pod_annotations_overwrite_allowed`](#the-keywords) to override pod annotations via the `.gitlab-ci.yml` file.

### Overwriting Container Resources

Additionally, Kubernetes CPU and memory allocations for requests and
limits for the build, helper and service containers can be overwritten
on the `.gitlab-ci.yml` file with the following variables:

``` yaml
 variables:
   KUBERNETES_CPU_REQUEST: 3
   KUBERNETES_CPU_LIMIT: 5
   KUBERNETES_MEMORY_REQUEST: 2Gi
   KUBERNETES_MEMORY_LIMIT: 4Gi
   KUBERNETES_EPHEMERAL_STORAGE_REQUEST: 512Mi
   KUBERNETES_EPHEMERAL_STORAGE_LIMIT: 1Gi

   KUBERNETES_HELPER_CPU_REQUEST: 3
   KUBERNETES_HELPER_CPU_LIMIT: 5
   KUBERNETES_HELPER_MEMORY_REQUEST: 2Gi
   KUBERNETES_HELPER_MEMORY_LIMIT: 4Gi
   KUBERNETES_HELPER_EPHEMERAL_STORAGE_REQUEST: 512Mi
   KUBERNETES_HELPER_EPHEMERAL_STORAGE_LIMIT: 1Gi

   KUBERNETES_SERVICE_CPU_REQUEST: 3
   KUBERNETES_SERVICE_CPU_LIMIT: 5
   KUBERNETES_SERVICE_MEMORY_REQUEST: 2Gi
   KUBERNETES_SERVICE_MEMORY_LIMIT: 4Gi
   KUBERNETES_SERVICE_EPHEMERAL_STORAGE_REQUEST: 512Mi
   KUBERNETES_SERVICE_EPHEMERAL_STORAGE_LIMIT: 1Gi
```

The values for these variables are restricted to what the max overwrite
for that resource has been set to.

## Define keywords in the configuration TOML

Each of the keywords can be defined in the `config.toml` file.

Here is an example `config.toml`:

```toml
concurrent = 4

[[runners]]
  name = "Kubernetes Runner"
  url = "https://gitlab.com/ci"
  token = "......"
  executor = "kubernetes"
  [runners.kubernetes]
    host = "https://45.67.34.123:4892"
    cert_file = "/etc/ssl/kubernetes/api.crt"
    key_file = "/etc/ssl/kubernetes/api.key"
    ca_file = "/etc/ssl/kubernetes/ca.crt"
    namespace = "gitlab"
    namespace_overwrite_allowed = "ci-.*"
    bearer_token_overwrite_allowed = true
    privileged = true
    cpu_limit = "1"
    memory_limit = "1Gi"
    service_cpu_limit = "1"
    service_memory_limit = "1Gi"
    helper_cpu_limit = "500m"
    helper_memory_limit = "100Mi"
    poll_interval = 5
    poll_timeout = 3600
    dns_policy = "cluster-first"
    [runners.kubernetes.node_selector]
      gitlab = "true"
    [runners.kubernetes.node_tolerations]
      "node-role.kubernetes.io/master" = "NoSchedule"
      "custom.toleration=value" = "NoSchedule"
      "empty.value=" = "PreferNoSchedule"
      "onlyKey" = ""
```

## Using volumes

As described earlier, volumes can be mounted in the build container.
At this time _hostPath_, _PVC_, _configMap_, _secret_, and _CSI_ volume types
are supported. Users can configure any number of volumes for each of
mentioned types.

Here is an example configuration:

```toml
concurrent = 4

[[runners]]
  # usual configuration
  executor = "kubernetes"
  [runners.kubernetes]
    [[runners.kubernetes.volumes.host_path]]
      name = "hostpath-1"
      mount_path = "/path/to/mount/point"
      read_only = true
      host_path = "/path/on/host"
    [[runners.kubernetes.volumes.host_path]]
      name = "hostpath-2"
      mount_path = "/path/to/mount/point_2"
      read_only = true
    [[runners.kubernetes.volumes.pvc]]
      name = "pvc-1"
      mount_path = "/path/to/mount/point1"
    [[runners.kubernetes.volumes.config_map]]
      name = "config-map-1"
      mount_path = "/path/to/directory"
      [runners.kubernetes.volumes.config_map.items]
        "key_1" = "relative/path/to/key_1_file"
        "key_2" = "key_2"
    [[runners.kubernetes.volumes.secret]]
      name = "secrets"
      mount_path = "/path/to/directory1"
      read_only = true
      [runners.kubernetes.volumes.secret.items]
        "secret_1" = "relative/path/to/secret_1_file"
    [[runners.kubernetes.volumes.empty_dir]]
      name = "empty-dir"
      mount_path = "/path/to/empty_dir"
      medium = "Memory"
    [[runners.kubernetes.volumes.csi]]
      name = "csi-volume"
      mount_path = "/path/to/csi/volume"
      driver = "my-csi-driver"
      [runners.kubernetes.volumes.csi.volume_attributes]
        size = "2Gi"
```

### Host Path volumes

[_HostPath_ volume](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath) configuration instructs Kubernetes to mount
a specified host path inside of the container. The volume can be configured with
following options:

| Option     | Type    | Required | Description |
|------------|---------|----------|-------------|
| name       | string  | yes      | The name of the volume |
| mount_path | string  | yes      | Path inside of container where the volume should be mounted |
| sub_path   | string  | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |
| host_path  | string  | no       | Host's path that should be mounted as volume. If not specified then set to the same path as `mount_path`. |
| read_only  | boolean | no       | Sets the volume in read-only mode (defaults to false) |

### PVC volumes

[_PVC_ volume](https://kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim) configuration instructs Kubernetes to use a _PersistentVolumeClaim_
that is defined in Kubernetes cluster and mount it inside of the container. The volume
can be configured with following options:

| Option     | Type    | Required | Description |
|------------|---------|----------|-------------|
| name       | string  | yes      | The name of the volume and at the same time the name of _PersistentVolumeClaim_ that should be used |
| mount_path | string  | yes      | Path inside of container where the volume should be mounted |
| read_only  | boolean | no       | Sets the volume in read-only mode (defaults to false) |
| sub_path   | string  | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |

### ConfigMap volumes

_ConfigMap_ volume configuration instructs Kubernetes to use a [_configMap_](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/)
that is defined in Kubernetes cluster and mount it inside of the container.

| Option     | Type    | Required | Description |
|------------|---------|----------|-------------|
| name       | string  | yes      | The name of the volume and at the same time the name of _configMap_ that should be used |
| mount_path | string  | yes      | Path inside of container where the volume should be mounted |
| read_only  | boolean | no       | Sets the volume in read-only mode (defaults to false) |
| sub_path   | string  | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |
| items      | `map[string]string` | no   | Key-to-path mapping for keys from the _configMap_ that should be used. |

When using _configMap_ volume, each key from selected _configMap_ will be changed into a file
stored inside of the selected mount path. By default all keys are present, _configMap's_ key
is used as file's name and value is stored as file's content. The default behavior can be
changed with `items` option.

`items` option is defining a mapping between key that should be used and path (relative
to volume's mount path) where _configMap's_ value should be saved. When using `items` option
**only selected keys** will be added to the volumes and all other will be skipped.

> **Notice**: If a non-existing key will be used then job will fail on Pod creation stage.

### Secret volumes

[_Secret_ volume](https://kubernetes.io/docs/concepts/storage/volumes/#secret) configuration instructs Kubernetes to use
a _secret_ that is defined in Kubernetes cluster and mount it inside of the container.

| Option     | Type    | Required | Description |
|------------|---------|----------|-------------|
| name       | string  | yes      | The name of the volume and at the same time the name of _secret_ that should be used |
| mount_path | string  | yes      | Path inside of container where the volume should be mounted |
| read_only  | boolean | no       | Sets the volume in read-only mode (defaults to false) |
| sub_path   | string  | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |
| items      | `map[string]string` | no   | Key-to-path mapping for keys from the _configMap_ that should be used. |

When using _secret_ volume each key from selected _secret_ will be changed into a file
stored inside of the selected mount path. By default all keys are present, _secret's_ key
is used as file's name and value is stored as file's content. The default behavior can be
changed with `items` option.

`items` option is defining a mapping between key that should be used and path (relative
to volume's mount path) where _secret's_ value should be saved. When using `items` option
**only selected keys** will be added to the volumes and all other will be skipped.

> **Notice**: If a non-existing key will be used then job will fail on Pod creation stage.

### Empty Dir volumes

[_emptyDir_ volume](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir) configuration instructs Kubernetes to mount an empty directory inside of the container.

| Option     | Type    | Required | Description |
|------------|---------|----------|-------------|
| name       | string  | yes      | The name of the volume |
| mount_path | string  | yes      | Path inside of container where the volume should be mounted |
| sub_path   | string  | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |
| medium     | string  | no       | "Memory" will provide a tmpfs, otherwise it defaults to the node disk storage (defaults to "") |

### CSI volumes

[_CSI_ volume](https://kubernetes.io/docs/concepts/storage/volumes/#csi) configuration instructs Kubernetes to use a custom CSI driver to mount an arbitrary storage system inside of the container.

| Option            | Type                | Required | Description |
|-------------------|---------------------|----------|-------------|
| name              | string              | yes      | The name of the volume |
| mount_path        | string              | yes      | Path inside of container where the volume should be mounted |
| driver            | string              | yes      | A string value that specifies the name of the volume driver to use. |
| fs_type           | string              | no       | A string value that specifies the name of the filesystem type (Ex. "ext4", "xfs", "ntfs".) |
| volume_attributes | `map[string]string` | no       | Key-value pair mapping for attributes of the CSI volume. |
| sub_path          | string              | no       | Mount a [sub-path](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) within the volume instead of the root. |
| read_only         | boolean             | no       | Sets the volume in read-only mode (defaults to false) |

## Using Security Context

[Pod security context](https://kubernetes.io/docs/concepts/policy/pod-security-policy/) configuration instructs executor to set a pod security policy on the build pod.

| Option              | Type     | Required | Description |
|---------------------|----------|----------|-------------|
| fs_group            | int      | no       | A special supplemental group that applies to all containers in a pod |
| run_as_group        | int      | no       | The GID to run the entrypoint of the container process |
| run_as_non_root     | boolean  | no       | Indicates that the container must run as a non-root user |
| run_as_user         | int      | no       | The UID to run the entrypoint of the container process |
| supplemental_groups | int list | no       | A list of groups applied to the first process run in each container, in addition to the container's primary GID |

Assigning a security context to pods provides security to your Kubernetes cluster. For this to work you'll need to provide a helper
image that conforms to the policy you set here.

[Read more about the helper image](../configuration/advanced-configuration.md#helper-image).
Example of building your own helper image:

```Dockerfile
ARG tag
FROM gitlab/gitlab-runner-helper:${tag}
RUN addgroup -g 59417 -S nonroot && \
    adduser -u 59417 -S nonroot -G nonroot
WORKDIR /home/nonroot
USER 59417:59417
```

This example creates a user and group called `nonroot` and sets the image to run as that user.

Example of setting pod security context in your `config.toml`:

```toml
concurrent = %(concurrent)s
check_interval = 30
  [[runners]]
    name = "myRunner"
    url = "gitlab.example.com"
    executor = "kubernetes"
    [runners.kubernetes]
      helper_image = "gitlab-registy.example.com/helper:latest"
      [runners.kubernetes.pod_security_context]
        run_as_non_root = true
        run_as_user = 59417
        run_as_group = 59417
        fs_group = 59417
```

## Using services

> - [Introduced](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/4470) in GitLab Runner 12.5.
> - [Introduced support for `alias`](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/4829) in GitLab Runner 12.9
> - [Introduced support for `command` and `entrypoint`](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/27173) in GitLab Runner 13.6.

Define a list of [services](https://docs.gitlab.com/ee/ci/services/).

```toml
concurrent = 1
check_interval = 30
  [[runners]]
    name = "myRunner"
    url = "gitlab.example.com"
    executor = "kubernetes"
    [runners.kubernetes]
      helper_image = "gitlab-registy.example.com/helper:latest"
      [[runners.kubernetes.services]]
        name = "postgres:12-alpine"
        alias = "db1"
      [[runners.kubernetes.services]]
        name = "registry.example.com/svc1"
        alias = "svc1"
        entrypoint = ["entrypoint.sh"]
        command = ["executable","param1","param2"]
```

## Using pull policies

The `pull_policy` parameter allows you to specify a single or multiple pull policies.

Using a single pull policy:

```toml
[runners.kubernetes]
  pull_policy = "never"
```

Using multiple pull policies:

```toml
[runners.kubernetes]
  # use multiple pull policies
  pull_policy = ["always", "if-not-present"]
```

The policy applies to both the build image, helper image and any services and controls how an image is fetched and updated.

Reading the documentation on Kubernetes [pull policies](https://v1-17.docs.kubernetes.io/docs/concepts/configuration/overview/#container-images) should
provide a good indication of when to use which.

Our naming convention is slightly different to Kubernetes. Below is a mapping from the Runner pull policy to the Kubernetes pull policy.

| Runner Pull Policy | Kubernetes Pull Policy | Description |
|--------------------|------------------------|-------------|
| _blank_ | _blank_ | Uses the default policy as specified by Kubernetes |
| `if-not-present` | `IfNotPresent` | the image is pulled only if it not already present on the node executing the job (see [security considerations](../security/index.md#usage-of-private-docker-images-with-if-not-present-pull-policy)) |
| `always` | `Always` | Always pulls the image is pulled every time the job is executed |
| `never` | `Never` | Never pulls the image and requires the node to already have it |

When multiple policies are defined, each policy is attempted until the image is obtained successfully. For example, using `[ always, if-not-present ]` allows the policy `if-not-present` to be used if `always` fails due to a temporary registry problem.

Beware of the [security considerations](../security/index.md#usage-of-private-docker-images-with-if-not-present-pull-policy) when using `if-not-present`.

## Adding extra host aliases

> [Introduced](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/2818) in GitLab 13.7.

This feature is available in Kubernetes 1.7+.

[Host aliases](https://kubernetes.io/docs/concepts/services-networking/add-entries-to-pod-etc-hosts-with-host-aliases/) configuration
instructs Kubernetes to add entries to `/etc/hosts` file inside of the container. The host aliases can be configured with the following options:

| Option     | Type     | Required | Description |
|------------|----------|----------|-------------|
| IP         | `string`   | yes      | The IP address you want to attach hosts too |
| Hostnames  | `[]string` | yes      | A list of host name aliases that will be attached to the IP |

Here is an example configuration:

```toml
concurrent = 4

[[runners]]
  # usual configuration
  executor = "kubernetes"
  [runners.kubernetes]
    [[runners.kubernetes.host_aliases]]
      ip = "127.0.0.1"
      hostnames = ["web1", "web2"]
    [[runners.kubernetes.host_aliases]]
      ip = "192.168.1.1"
      hostnames = ["web14", "web15"]
```

## Using Affinity

### Node Affinity

> [Introduced](https://gitlab.com/gitlab-org/gitlab-runner/-/merge_requests/2324) in GitLab Runner 13.4.

Define a list of [node affinities](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity) to be added to a pod specification at build time.

```toml
concurrent = 1
  [[runners]]
    name = "myRunner"
    url = "gitlab.example.com"
    executor = "kubernetes"
    [runners.kubernetes]
      [runners.kubernetes.affinity]
        [runners.kubernetes.affinity.node_affinity]
          [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution]]
            weight = 100
            [runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference]
              [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_expressions]]
               key = "cpu_speed"
               operator = "In"
               values = ["fast"]
              [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_expressions]]
               key = "mem_speed"
               operator = "In"
               values = ["fast"]
          [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution]]
            weight = 50
            [runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference]
              [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_expressions]]
                key = "core_count"
                operator = "In"
                values = ["high", "32"]
              [[runners.kubernetes.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_fields]]
               key = "cpu_type"
               operator = "In"
               values = ["arm64"]
        [runners.kubernetes.affinity.node_affinity.required_during_scheduling_ignored_during_execution]
          [[runners.kubernetes.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_terms]]
            [[runners.kubernetes.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_terms.match_expressions]]
              key = "kubernetes.io/e2e-az-name"
              operator = "In"
              values = [
                "e2e-az1",
                "e2e-az2"
              ]
```

## Pod's DNS Config

> - [Introduced](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/6562) in GitLab Runner 13.7.

[Pod's DNS Config](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-dns-config) gives users more control on the DNS settings of the created build Pods.

```toml
concurrent = 1
check_interval = 30
[[runners]]
  name = "myRunner"
  url = "https://gitlab.example.com"
  token = "__REDACTED__"
  executor = "kubernetes"
  [runners.kubernetes]
    image = "alpine:latest"
    [dns_config]
      nameservers = [
        "1.2.3.4",
      ]
      searches = [
        "ns1.svc.cluster-domain.example",
        "my.dns.search.suffix",
      ]

      [[dns_config.options]]
        name = "ndots"
        value = "2"

      [[dns_config.options]]
        name = "edns0"
```

| Option       | Type                        | Required | Description |
|--------------|-----------------------------|----------|-------------|
| nameservers  | `string` list               | no       | A list of IP addresses that will be used as DNS servers for the Pod |
| options      | `KubernetesDNSConfigOption` | no       | A optional list of objects where each object may have a name property (required) and a value property (optional) |
| searches     | `string` list               | no       | A list of DNS search domains for hostname lookup in the Pod |

### KubernetesDNSConfigOption

| Option       | Type      | Required | Description |
|--------------|-----------|----------|-------------|
| name         | `string`  | yes      | Configuration option name  |
| value        | `*string` | no       | Configuration option value |

## Capabilities configuration

[Kubernetes allows](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-capabilities-for-a-container)
to configure different [Linux capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html) that should be
added or dropped from a container.

GitLab Runner supports configuration of capabilities with the `cap_add` and `cap_drop` keywords in the `[runners.kubernetes]`
section of the configuration file. By default, GitLab Runner provides
a [list of capabilities](#default-list-of-dropped-capabilities) that should be dropped.

Because the default list of dropped capabilities can intersect with user-defined capabilities, the following rules are
applied to determine the final list:

1. User-defined `cap_drop` has priority over user-defined `cap_add`. If
   you define the same capability in both settings, only the one from `cap_drop` is passed
   to the container.

1. User-defined `cap_add` has priority over the default list of dropped capabilities.
   If you want to add the capability that is dropped by default, explicitly add it to
   `cap_add`.

The final list of capabilities is added to all containers in the job's pod.

A few notes:

- Remove the `CAP_` prefix from capability identifiers passed to the container configuration.
  For example, if you want to add or drop the `CAP_SYS_TIME` capability,
  in the configuration file, set a `SYS_TIME` string for `cap_add` or `cap_drop`.

- The owner of the Kubernetes cluster
  [can define a PodSecurityPolicy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#capabilities),
  where specific capabilities are allowed, restricted, or added by default.
  These rules take precedence over any user-defined configuration.

  Container runtimes can also define a default list of capabilities that you can add,
  like those seen in
  [Docker](https://github.com/moby/moby/blob/19.03/oci/defaults.go#L14-L32)
  or [containerd](https://github.com/containerd/containerd/blob/v1.4.0/oci/spec.go#L93-L110).

### Default list of dropped capabilities

GitLab Runner tries to drop these capabilities by default. If any of them are required for the job
to be executed properly, you should explicitly add the capability with the `cap_add` setting:

<!-- kubernetes_default_cap_drop_list_start -->
- `NET_RAW`

<!-- kubernetes_default_cap_drop_list_end -->

### Example configuration

```toml
concurrent = 1
check_interval = 30
[[runners]]
  name = "myRunner"
  url = "gitlab.example.com"
  executor = "kubernetes"
  [runners.kubernetes]
    # ...
    cap_add = ["SYS_TIME", "IPC_LOCK"]
    cap_drop = ["SYS_ADMIN"]
    # ...
```

## Using Docker in your builds

There are a couple of caveats when using Docker in your builds while running on
a Kubernetes cluster. Most of these issues are already discussed in the
[**Using Docker Build**](https://docs.gitlab.com/ee/ci/docker/using_docker_build.html)
section of the GitLab CI
documentation but it is worthwhile to revisit them here as you might run into
some slightly different things when running this on your cluster.

### Exposing `/var/run/docker.sock`

Exposing your host's `/var/run/docker.sock` into your build container, using the
`runners.kubernetes.volumes.host_path` option, brings the same risks with it as
always. That node's containers are accessible from the build container and
depending if you are running builds in the same cluster as your production
containers it might not be wise to do that.

### Using `docker:dind`

Running the `docker:dind` also known as the `docker-in-docker` image is also
possible but sadly needs the containers to be run in privileged mode.
If you're willing to take that risk other problems will arise that might not
seem as straight forward at first glance. Because the Docker daemon is started
as a `service` usually in your `.gitlab-ci.yaml` it will be run as a separate
container in your Pod. Basically containers in Pods only share volumes assigned
to them and an IP address by which they can reach each other using `localhost`.
`/var/run/docker.sock` is not shared by the `docker:dind` container and the `docker`
binary tries to use it by default.

To overwrite this and make the client use TCP to contact the Docker daemon,
in the other container, be sure to include the environment variables of
the build container:

- `DOCKER_HOST=tcp://localhost:2375` for no TLS connection.
- `DOCKER_HOST=tcp://localhost:2376` for TLS connection.

Make sure to configure those properly. As of Docker 19.03, TLS is enabled by
default but it requires mapping
certificates to your client. You can enable non-TLS connection for DIND or
mount certificates as described in
[**Use Docker In Docker Workflow with Docker executor**](https://docs.gitlab.com/ee/ci/docker/using_docker_build.html#use-docker-in-docker-workflow-with-docker-executor)

### Not supplying Git

Do *not* try to use an image that doesn't supply Git and add the `GIT_STRATEGY=none`
environment variable for a job that you think doesn't need to do a fetch or clone.
Because Pods are ephemeral and do not keep state of previously run jobs your
checked out code will not exist in both the build and the Docker service container.
Errors you might run into are things like `could not find git binary` and
the Docker service complaining that it cannot follow some symlinks into your
build context because of the missing code.

### Resource separation

In both the `docker:dind` and `/var/run/docker.sock` cases the Docker daemon
has access to the underlying kernel of the host machine. This means that any
`limits` that had been set in the Pod will not work when building Docker images.
The Docker daemon will report the full capacity of the node regardless of
the limits imposed on the Docker build containers spawned by Kubernetes.

One way to help minimize the exposure of the host's kernel to any build container
when running in privileged mode or by exposing `/var/run/docker.sock` is to use
the `node_selector` option to set one or more labels that have to match a node
before any containers are deployed to it. For example build containers may only run
on nodes that are labeled with `role=ci` while running all other production services
on other nodes. Further separation of build containers can be achieved using node
[taints](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/).
This will disallow other pods from scheduling on the same nodes as the
build pods without extra configuration for the other pods.

### Using kaniko

Another approach for building Docker images inside a Kubernetes cluster is using [kaniko](https://github.com/GoogleContainerTools/kaniko).
kaniko:

- Allows you to build images without privileged access.
- Works without the Docker daemon.

For more information, see [Building images with kaniko and GitLab CI/CD](https://docs.gitlab.com/ee/ci/docker/using_kaniko.html).

## Job execution

At the moment we are using `kube exec` to run the scripts, which relies on
having a stable network connection between the Runner and the pod for the duration of the command.
This leads to problems like [Job marked as success midway](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/4119).
If you are experiencing this problem turn off the feature flag [`FF_USE_LEGACY_KUBERNETES_EXECUTION_STRATEGY`](../configuration/feature-flags.md#available-feature-flags)
to use `kube attach` for script execution, which is more stable.

We are rolling this out slowly and have plans to enable the `kube attach` behavior by default in future release, please follow [#10341](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/10341) for updates.

## Troubleshooting

The following errors are commonly encountered when using the Kubernetes executor.

### `Build failed (system failure): timedout waiting for pod to start`

If the cluster cannot schedule the build pod before the timeout defined by `poll_timeout`, the build pod returns an error. The [Kubernetes Scheduler](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-lifetime) should be able to delete it.

To fix this issue, increase the `poll_timeout` value in your `config.toml` file.
