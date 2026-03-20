# 🏗️ Infrastructure Guide - OnChainHealthMonitor

> Complete guide for provisioning and deploying OnChainHealthMonitor to Google Kubernetes Engine (GKE) using Terraform and Helm.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Terraform - Provisioning GKE](#2-terraform--provisioning-gke)
3. [Helm - Deploying Services](#3-helm--deploying-services)
4. [Kubernetes - Operating the Cluster](#4-kubernetes--operating-the-cluster)
5. [Complete Deployment Workflow](#5-complete-deployment-workflow)

---

## 1. Overview

### Local vs Production

OnChainHealthMonitor runs in two environments:

| Environment | Orchestrator | How to start |
|-------------|-------------|--------------|
| **Local (dev)** | `docker-compose` | `docker-compose up --build` |
| **Production** | GKE (Kubernetes) | Terraform + Helm (this guide) |

Local development uses Docker Compose to spin up all services and the observability stack on a single machine. Production uses a real Kubernetes cluster on GCP - with autoscaling, workload identity, shielded nodes, and proper resource isolation.

### Infrastructure Components

```
GCP Project
└── VPC Network (europe-west1)
    └── Subnet (10.0.0.0/16)
        └── GKE Cluster
            └── Node Pool (autoscaling: 1–5 nodes)
                └── Namespace: onchain-health-monitor
                    ├── Deployment: api          (HPA: 2–10 replicas)
                    ├── Deployment: collector
                    ├── Deployment: analyzer
                    ├── Deployment: notifier
                    ├── ConfigMaps (per service)
                    ├── Services (ClusterIP)
                    ├── ServiceMonitors (Prometheus scraping)
                    └── prometheus-config (ConfigMap)
```

### GKE vs k3s

| | GKE | k3s |
|---|---|---|
| **Use case** | Real production deployment | Zero-cost local alternative |
| **Cost** | ~$75/month (1 × e2-medium) | Free |
| **Workload Identity** | ✅ Yes | ❌ No |
| **Autoscaling** | ✅ Cluster + HPA | ✅ HPA only |
| **Setup time** | ~10 min (Terraform) | ~5 min |

The same Helm charts work on both GKE and k3s. Only the Terraform step differs.

---

## 2. Terraform - Provisioning GKE

### What Terraform Provisions

Running `terraform apply` creates:

1. **VPC Network** - a dedicated GCP VPC with a private subnet in `europe-west1`
2. **Subnet** - `10.0.0.0/16` CIDR, with secondary ranges for Pod and Service IPs
3. **GKE Cluster** - regional cluster with:
   - Workload Identity enabled (pods can act as GCP service accounts)
   - Shielded nodes (Secure Boot + integrity monitoring)
   - Node autoscaling (1–5 nodes, `e2-medium`)
   - Private networking through the VPC module

### Module Structure

```
infra/terraform/
├── main.tf                  # Root: calls modules, wires outputs
├── variables.tf             # Input variables (project_id, region, cluster_name)
├── outputs.tf               # Exports: cluster endpoint, kubeconfig command
├── terraform.tfvars.example # Template - copy to terraform.tfvars
├── modules/
│   ├── networking/          # VPC, subnet, secondary IP ranges
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── gke/                 # GKE cluster, node pool, workload identity
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
```

The **networking module** is called first and outputs the VPC and subnet self-links. The **gke module** consumes those links to place the cluster inside the correct network.

### Prerequisites

| Tool | Required version | Install |
|------|-----------------|---------|
| `gcloud` CLI | Latest | https://cloud.google.com/sdk/docs/install |
| `terraform` | ≥ 1.7.0 | https://developer.hashicorp.com/terraform/install |
| GCP project | With billing enabled | https://console.cloud.google.com |

You also need the following GCP APIs enabled on your project:

```bash
gcloud services enable container.googleapis.com
gcloud services enable compute.googleapis.com
gcloud services enable iam.googleapis.com
```

Authenticate:

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### Deploying

```bash
cd infra/terraform

# 1. Copy the example vars file and fill in your project ID
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars - set project_id = "your-gcp-project-id"

# 2. Initialise Terraform (downloads providers)
terraform init

# 3. Preview what will be created
terraform plan

# 4. Apply (type "yes" when prompted)
terraform apply
```

> ⏱️ `terraform apply` takes approximately **8–12 minutes** to provision the GKE cluster.

### Getting kubectl Access

After `terraform apply` completes, the output includes a `gcloud` command to configure kubectl:

```bash
# From the Terraform output:
gcloud container clusters get-credentials <cluster-name> \
  --region europe-west1 \
  --project <your-project-id>

# Verify connection
kubectl get nodes
```

### State Management

By default, Terraform state is stored locally in `terraform.tfstate`. For team use, configure a **GCS backend** in `main.tf`:

```hcl
terraform {
  backend "gcs" {
    bucket = "your-terraform-state-bucket"
    prefix = "onchainhealthmonitor/state"
  }
}
```

The `terraform.tfvars.example` file includes a stub comment for this.

### Destroying

> ⚠️ This destroys the GKE cluster and all workloads running on it.

```bash
terraform destroy
```

---

## 3. Helm - Deploying Services

### What Helm Does

Kubernetes resources are defined in YAML. Without Helm, you'd have dozens of nearly-identical YAML files for each service - and no way to customise values (like image tags) per environment. Helm solves this with a **template engine**:

```
templates/ (with {{ .Values.x }})  +  values.yaml  =  Kubernetes YAML
```

You define the structure once. Values change per environment. One `helm upgrade` rolls out all services atomically.

### Chart Structure

OnChainHealthMonitor uses an **umbrella chart** pattern: one parent chart that depends on four per-service subcharts.

```
infra/helm/
└── onchain-health-monitor/          # Umbrella chart
    ├── Chart.yaml                # Declares dependencies on subcharts
    ├── values.yaml               # Global defaults (image registry, tag, etc.)
    └── charts/                   # Subchart tarballs (populated by helm dep update)
        ├── api/                  # Per-service chart
        │   ├── Chart.yaml
        │   ├── values.yaml
        │   └── templates/
        │       ├── deployment.yaml
        │       ├── service.yaml
        │       ├── hpa.yaml
        │       └── configmap.yaml
        ├── collector/
        ├── analyzer/
        └── notifier/
```

Each per-service chart contains:
- **Deployment** - runs the container from `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest`
- **Service** - exposes the pod internally on its port (ClusterIP)
- **HPA** - Horizontal Pod Autoscaler (CPU-based scaling)
- **ConfigMap** - injects environment-specific configuration

### Deploying All Services

```bash
cd infra/helm

# 1. Download/update subchart dependencies
helm dep update onchain-health-monitor

# 2. Install all four services in one go
helm install onchain-health-monitor ./onchain-health-monitor \
  --namespace onchain-health-monitor \
  --create-namespace
```

This creates the `onchain-health-monitor` namespace and deploys all four services simultaneously.

### Upgrading a Service

To roll out a new image tag for the `api` service:

```bash
helm upgrade onchain-health-monitor ./onchain-health-monitor \
  --set api.image.tag=sha-abc1234
```

To upgrade with a full values override file (e.g., for production):

```bash
helm upgrade onchain-health-monitor ./onchain-health-monitor \
  --values values-production.yaml
```

### Checking Status

```bash
# Helm release status
helm status onchain-health-monitor -n onchain-health-monitor

# Pod status
kubectl get pods -n onchain-health-monitor

# Expected output:
# NAME                         READY   STATUS    RESTARTS   AGE
# api-xxxxxxxxxx-xxxxx         1/1     Running   0          2m
# collector-xxxxxxxxxx-xxxxx   1/1     Running   0          2m
# analyzer-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
# notifier-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
```

### Rolling Back

If a deployment goes wrong, roll back to the previous Helm revision:

```bash
# List revisions
helm history onchain-health-monitor -n onchain-health-monitor

# Roll back to revision 1
helm rollback onchain-health-monitor 1 -n onchain-health-monitor
```

### Uninstalling

```bash
helm uninstall onchain-health-monitor -n onchain-health-monitor
```

---

## 4. Kubernetes - Operating the Cluster

### Namespace

All workloads run in the `onchain-health-monitor` namespace. This isolates them from system pods and makes RBAC policies easier to scope.

The namespace is defined in `infra/k8s/namespace.yaml` and is also created automatically by `helm install --create-namespace`.

### Checking Service Health

```bash
# All pods in the namespace
kubectl get pods -n onchain-health-monitor

# Tail logs for the api service
kubectl logs -f deployment/api -n onchain-health-monitor

# Describe a pod for events/errors
kubectl describe pod <pod-name> -n onchain-health-monitor

# Local access to the api service (bypasses ingress)
kubectl port-forward svc/api 8080:8080 -n onchain-health-monitor
```

Then in another terminal:
```bash
curl http://localhost:8080/api/v1/protocols
```

### Horizontal Pod Autoscaler (HPA)

The `api` service has an HPA configured in its Helm chart. It scales based on CPU utilisation:

- **Minimum replicas:** 2
- **Maximum replicas:** 10
- **Target CPU utilisation:** 70%

When traffic increases and average CPU across api pods exceeds 70%, Kubernetes automatically adds more replicas (up to 10). When traffic drops, it scales back down to 2.

```bash
# View current HPA state
kubectl get hpa -n onchain-health-monitor

# Example output:
# NAME   REFERENCE         TARGETS   MINPODS   MAXPODS   REPLICAS
# api    Deployment/api    45%/70%   2         10        2
```

### ServiceMonitors (Prometheus Operator)

`infra/k8s/` contains four `ServiceMonitor` resources - one per service. A `ServiceMonitor` is a custom resource (CRD) that tells the **Prometheus Operator** which services to scrape for metrics.

**Why ServiceMonitors instead of `prometheus.yml`?**

In Kubernetes, pod IPs change constantly. Hard-coding endpoints in `prometheus.yml` doesn't work. The Prometheus Operator watches ServiceMonitor resources and automatically updates Prometheus's scrape targets as pods come and go.

**Prerequisite:** The Prometheus Operator must be installed in the cluster. The `infra/k8s/prometheus-config.yaml` ConfigMap configures scrape intervals (10s) and retention for the monitoring stack.

```bash
# View ServiceMonitors
kubectl get servicemonitors -n onchain-health-monitor

# Expected output:
# NAME        AGE
# api         5m
# collector   5m
# analyzer    5m
# notifier    5m
```

---

## 5. Complete Deployment Workflow

Step-by-step from zero to all services running on GKE.

### Step 1 - Provision GKE with Terraform

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: set project_id to your GCP project

terraform init
terraform plan   # Review what will be created
terraform apply  # Takes ~10 minutes
```

### Step 2 - Configure kubectl

```bash
# Use the command from terraform output
gcloud container clusters get-credentials <cluster-name> \
  --region europe-west1 \
  --project <your-project-id>

# Verify
kubectl get nodes
# NAME                                STATUS   ROLES    AGE   VERSION
# gke-onchain-monitor-pool-xxxxx         Ready    <none>   5m    v1.29.x
```

### Step 3 - Create the Namespace

The namespace is created automatically in Step 4. If you need it separately (e.g., to apply ServiceMonitors first):

```bash
kubectl apply -f infra/k8s/namespace.yaml
```

### Step 4 - Deploy with Helm

```bash
cd infra/helm

# Update subchart dependencies
helm dep update onchain-health-monitor

# Deploy all four services
helm install onchain-health-monitor ./onchain-health-monitor \
  --namespace onchain-health-monitor \
  --create-namespace
```

### Step 5 - Apply Kubernetes Manifests

```bash
# Apply Prometheus ConfigMap and ServiceMonitors
kubectl apply -f infra/k8s/ -n onchain-health-monitor
```

### Step 6 - Verify All Pods Running

```bash
kubectl get pods -n onchain-health-monitor --watch
```

Wait until all four pods show `Running` and `1/1` READY. This typically takes 30–60 seconds as images are pulled from GHCR.

```bash
# Check services are reachable internally
kubectl get svc -n onchain-health-monitor
# NAME        TYPE        CLUSTER-IP     PORT(S)
# api         ClusterIP   10.100.x.x     8080/TCP
# collector   ClusterIP   10.100.x.x     8081/TCP
# analyzer    ClusterIP   10.100.x.x     8082/TCP
# notifier    ClusterIP   10.100.x.x     8083/TCP
```

### Step 7 - Test Locally via Port-Forward

```bash
# Forward the api service to your local machine
kubectl port-forward svc/api 8080:8080 -n onchain-health-monitor
```

In another terminal:

```bash
# Health check
curl http://localhost:8080/health
# {"status":"ok"}

# List protocols
curl http://localhost:8080/api/v1/protocols

# Metrics
curl http://localhost:8080/metrics
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `ImagePullBackOff` | GHCR image not accessible | Check image name: `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest` |
| Pod stuck in `Pending` | Insufficient node capacity | Check `kubectl describe pod <name>` for resource pressure; wait for autoscaler |
| `CrashLoopBackOff` | Service misconfigured | Check `kubectl logs <pod-name>` for startup errors |
| ServiceMonitor not working | Prometheus Operator missing | Install `prometheus-community/kube-prometheus-stack` chart |
| `helm dep update` fails | No internet from cluster | Run locally, not in cluster |

---

*For local development setup, see [LOCAL_SETUP.md](./LOCAL_SETUP.md). For CI/CD pipeline details, see [CI_CD_GUIDE.md](./CI_CD_GUIDE.md). For architecture decisions, see [../architecture/DECISIONS.md](../architecture/DECISIONS.md).*
