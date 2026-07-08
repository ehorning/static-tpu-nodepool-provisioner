# Static Nodepool Provisioner

A lightweight Kubernetes controller designed to pre-provision and manage GKE TPU node pools based on a static configuration. 

This provisioner is specifically optimized for workloads utilizing **gSC (Google Supercomputing) reservations** and **superslicing** on Google Kubernetes Engine. It ensures highly predictable, resilient, and state-of-the-art life cycle management of static hardware capacities.

---

## Key Features

- **Static Pre-Provisioning:** Watches a central `ConfigMap` for declarations of reserved capacity blocks (GSC blocks) and spawns node pools matching these reservations.
- **In-Use Safety Guards:** Automatically detects whether a static node pool is actively consumed by any running `Slice` workloads, preventing accidental deletion or modification.
- **Self-Healing State Retries:** Detects node pools that have entered an `ERROR` state and triggers fully managed tear-down and retry cycles.
- **Precise Hash Hashing:** Computes cryptographic hashes of configuration specs to cleanly manage rolling recreations (updates) without unnecessary thrashing.

---

## Configuration

The **Static Nodepool Provisioner** is entirely configured via a single Kubernetes `ConfigMap` named `tpu-provisioner-static-nodepools-config` in the same namespace as the controller (by default: `tpu-provisioner-system`).

The controller is created in the namespace `tpu-provisioner-system` by default and so both the controller and the validating webhook watches for config maps created in this namespace.

### Configuration Specification Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tpu-provisioner-static-nodepools-config
  namespace: tpu-provisioner-system
data:
  reservations: |
    - name: "my-gsc-reservation"
      gscSubblocks:
        - block: "block-1"
          subblocks: "0001-0004"
          nodepoolConfig:
            nodepoolPrefix: "example-1"
            machineType: "tpu7x-standard-4t"
            topology: "4x4x4"
            nodeCount: 16
        - block: "block-1"
          subblocks: "0005"
          nodepoolConfig:
            nodepoolPrefix: "example-2"
            machineType: "tpu7x-standard-4t"
            topology: "2x2x4"
            nodeCount: 8
        - block: "block-2"
          subblocks: "0001-0002"
          nodepoolConfig:
            nodepoolPrefix: "example-3"
  # Default/fallback config map applied to all nodepools unless overridden at the subblock level
  defaultNodepoolConfig: |
    nodepoolPrefix: "example"
    machineType: "tpu7x-standard-4t"
    accelerator: "tpu7x"
    topology: "4x4x4"
    nodeCount: 16
    nodeLabels:
      my-custom-key: "my-custom-value"
    shieldedIntegrityMonitoring: true
    shieldedSecureBoot: true
    maxPodsPerNode: 15
    enableAutorepair: true
```

### Parameter Explanations

#### `reservations`
A list of GSC reservations each containing a list of subblocks:
- **`name`**: The exact GCP reservation name.
- **`gscSubblocks`**: The list of subblocks.
  - **`block`**: The GSC block identifier.
  - **`subblocks`**: Subblock index range to target (e.g., `"0001-0004"` or a single index `"0002"`).
  - **`nodepoolConfig`** *(Optional)*: Override defaults or define custom GKE NodePool configurations on a per-subblock-range basis. Fields specified here override or augment the root-level fallback `defaultNodepoolConfig` field-by-field.

Specifying an empty reservations list will result in all node pools managed by the controller being deleted.

#### `defaultNodepoolConfig` (and `nodepoolConfig` under subblocks)
Configuration attributes applied as defaults/fallbacks (or on a per-subblock level):
- **`nodepoolPrefix`** *(Optional)*: Prefix for created node pool names. If omitted, falls back to the default config prefix, then to using the block name.
- **`machineType`**: The GKE machine type to use (e.g., `tpu7x-standard-4t`).
- **`accelerator`**: The TPU accelerator type.
- **`topology`**: Network physical topology (e.g., `4x4x4`).
- **`nodeCount`**: Count of GKE nodes in the pool.
- **`nodeLabels`**: Set of labels to inject into the nodes.
- **`shieldedIntegrityMonitoring`** *(Optional)*: Enable/disable GKE Shielded integrity monitoring.
- **`shieldedSecureBoot`** *(Optional)*: Enable/disable Secure Boot. Takes precedence over global defaults.
- **`maxPodsPerNode`** *(Optional)*: Max number of Pods per node. Takes precedence over global environment variables.
- **`enableAutorepair`** *(Optional)*: Enable GKE auto-repair.

---

### How to guides

#### Create a TPU node pool targeting single subblock

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: tpu-provisioner-static-nodepools-config
  namespace: tpu-provisioner-system
data:
  reservations: |
    - name: "my-gsc-reservation"
      gscSubblocks:
        - block: "block-1"
          subblocks: "0001"
          nodepoolConfig:
            nodepoolPrefix: "example-1"
            machineType: "tpu7x-standard-4t"
            topology: "4x4x4"
            nodeCount: 16
  # Default/fallback config map applied to all nodepools unless overridden at the subblock level
  defaultNodepoolConfig: |
    nodepoolPrefix: "example"
    machineType: "tpu7x-standard-4t"
    accelerator: "tpu7x"
    topology: "4x4x4"
    nodeCount: 16
    nodeLabels:
      my-custom-key: "my-custom-value"
    shieldedIntegrityMonitoring: true
    shieldedSecureBoot: true
    maxPodsPerNode: 15
    enableAutorepair: true
```

#### Create TPU node pools targeting a list of subblocks

A separate node pool is created for each subblock.
The following command will result in the creation of the node pools: "example-1-0001", "example-1-0002", "example-1-0003", "example-1-0004"

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: tpu-provisioner-static-nodepools-config
  namespace: tpu-provisioner-system
data:
  reservations: |
    - name: "my-gsc-reservation"
      gscSubblocks:
        - block: "block-1"
          subblocks: "0001-0004"
          nodepoolConfig:
            nodepoolPrefix: "example-1"
            machineType: "tpu7x-standard-4t"
            topology: "4x4x4"
            nodeCount: 16
  # Default/fallback config map applied to all nodepools unless overridden at the subblock level
  defaultNodepoolConfig: |
    nodepoolPrefix: "example"
    machineType: "tpu7x-standard-4t"
    accelerator: "tpu7x"
    topology: "4x4x4"
    nodeCount: 16
    nodeLabels:
      my-custom-key: "my-custom-value"
    shieldedIntegrityMonitoring: true
    shieldedSecureBoot: true
    maxPodsPerNode: 15
    enableAutorepair: true
```

#### Delete all node pools managed by the controller

To delete all node pools managed by the controller, empty the reservations field in the configmap.
NOTE: You must provide a defaultNodepoolConfig for the controller to consider the configmap to be valid if reservations is empty.

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: tpu-provisioner-static-nodepools-config
  namespace: tpu-provisioner-system
data:
  reservations: "[]"
  defaultNodepoolConfig: |
    nodepoolPrefix: "example"
    machineType: "tpu7x-standard-4t"
    accelerator: "tpu7x"
    topology: "4x4x4"
    nodeCount: 16
    nodeLabels:
      my-custom-key: "my-custom-value"
    shieldedIntegrityMonitoring: true
    shieldedSecureBoot: true
    maxPodsPerNode: 15
    enableAutorepair: true
```

---

## Life Cycle Reconcile Safeguards

### In-Use Safety Lock
Nodepools created by this provisioner are labeled with `tpu-provisioner-static-nodepool=true`. 
Before any node pool deletion or modification is executed:
1. The provisioner fetches all **Slice** custom resources in the cluster and maps their in-use partitions.
2. It examines the node pool's nodes to see if they hold partition label identifiers locked by these active Slices.
3. If an active lease is found, deletion or configuration recreation is **deferred** (skipped) until the workload releases the capacity, guaranteeing zero service interruption.

### Self-Healing Error Mitigation
If a node pool build fails and falls into GKE `ERROR` status (e.g., because of cloud quota limits or transient system outages), the manager automatically identifies the problem, deletes the failing capacity, and schedules an organic rebuild.

---

## Deployment

The static TPU nodepool provisioner is designed to be deployed into the `tpu-provisioner-system` namespace. You can deploy it using the provided manifests in the `deploy/` directory.

### Prerequisites

1. Ensure you have a cluster running with Workload Identity enabled, as the controller needs permission to manage NodePools via the GCP API. You also need an Artifact Registry or Docker Hub repository to host the built image.
2. Ensure that the cluster has Slice Controller enabled on it as we watch for Slices to determine if we can delete a node as a safety check. Follow https://docs.cloud.google.com/kubernetes-engine/docs/how-to/use-gke-dynamic-slicing#enable_the_slice_controller for instructions on how to enable Slice controller and ensure you have the `slices.accelerator.gke.io` CRD present in the cluster.
3. Since this controller uses a Validating Webhook to reject invalid ConfigMaps, you must install [cert-manager](https://cert-manager.io/docs/installation/) in your cluster to automatically provision TLS certificates for the webhook server.

NOTE: Your cluster must have access to the internet to pull the images for cert-manager or you must edit the configuration to use private images from repositories that your cluster can access.

### 1. Build and Push the Image

Build and push the provisioner image using the provided Makefile targets:

```bash
# Build the image
make docker-build IMG=gcr.io/your-project/static-np-provisioner:latest

# Push the image
make docker-push IMG=gcr.io/your-project/static-np-provisioner:latest
```

### 2. Configure Manifests

Before applying the manifests, you must configure your environment-specific values in `deploy/deployment.yaml`:
1. Replace `<image>` with your pushed image tag (e.g. `gcr.io/your-project/static-np-provisioner:latest`).
2. Replace `<gsa>` with the Google Service Account email that has permissions to manage GKE NodePools.
3. Update the `GCP_ZONE` environment variable if your cluster is in a different zone.
4. If you've updated the namespace that the controller is being deployed to, update the namespace that the webhook is watching by updating `webhook[0].namespaceSelector.matchLabels.kubernetes.io/metadata.name` field in `deploy/webhook.yaml`.

### 3. Configure Workload Identity

The provisioner uses GKE Workload Identity to authenticate with GCP APIs. You must bind the Kubernetes Service Account (KSA) to your Google Service Account (GSA) so the provisioner is granted access.

If the cluster hasn't already enabled Workload Identity, follow the instructions here: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity

Run the following command to create the IAM policy binding (replace `<gsa>` and `<project-id>`):

```bash
gcloud iam service-accounts add-iam-policy-binding <gsa> \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:<project-id>.svc.id.goog[tpu-provisioner-system/tpu-provisioner-controller-manager]"
```

*Note: The GSA must have the `roles/container.clusterAdmin` role (or equivalent permissions to create and delete GKE NodePools).*

### 4. Apply Manifests

Once the manifests are updated, apply the RBAC permissions, deployment, webhook, and an initial ConfigMap:

```bash
kubectl apply -f config/rbac/role.yaml
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/certmanager.yaml
kubectl apply -f deploy/webhook.yaml
kubectl apply -f deploy/configmap.yaml
```

---

## Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PROVIDER` | `gke` | Cloud backend (`gke` or `mock`). |
| `POD_NAMESPACE` | `default` | Namespace where the configmap resides. |
| `GCP_PROJECT_ID` | *Auto* | GCP Project ID (inferred on GCE). |
| `GCP_CLUSTER` | *Auto* | GKE Cluster Name (inferred on GCE). |
| `GCP_CLUSTER_LOCATION` | *Auto* | GKE Cluster Region/Zone (inferred on GCE). |
| `GCP_ZONE` | *Auto* | Compute Zone where nodes are spawned. |
| `GCP_NODE_SERVICE_ACCOUNT` | *None* | Service account identity for node pools. |
| `GCP_NODE_TAGS` | *None* | Comma-separated GCE instance network tags. |
| `GCP_NODE_SECURE_BOOT` | `true` | Set Secure Boot as global default. |
| `GCP_NODE_ADDITIONAL_NETWORKS`| *None* | Comma-separated extra VPC and subnets (`vpc:subnet,...`). |
| `GKE_MAX_PODS_PER_NODE` | `16` | Global maximum pods per node configuration. |
| `STATIC_NODEPOOL_CREATE_CONCURRENCY` | `3` | Parallel node pool creation limit. |
| `STATIC_NODEPOOL_DELETE_CONCURRENCY` | `3` | Parallel node pool deletion limit. |
| `STATIC_NODEPOOL_CREATE_TIMEOUT` | `10m` | Time to wait for GKE node pool operations. |

---

## Development

### Running Locally

Set the following environment variables and then run `make run`:

```bash
export GCP_PROJECT_ID="your-gcp-project"
export GCP_CLUSTER_LOCATION="your-cluster-region-or-zone"
export GCP_CLUSTER="your-cluster-name"
export GCP_ZONE="your-compute-zone" # (Zone where nodes should reside)
```

### Building the Binary
To compile the standalone static nodepool provisioner binary:
```bash
make build
```
This produces the executable inside `bin/manager`.

### Running Tests

To run the unit tests:
```bash
make test-unit
```

To run the integration suite (requires [Envtest](https://book.kubebuilder.io/reference/envtest.html)):
```bash
make test-integration
```

To run the full verification pipeline:
```bash
make test
```
