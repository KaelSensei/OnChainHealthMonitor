variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region where the cluster will be created"
  type        = string
}

variable "zone" {
  description = "GCP zone (used for zonal resources)"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
}

variable "cluster_name" {
  description = "Base name for the GKE cluster"
  type        = string
}

variable "network" {
  description = "VPC network name to attach the cluster to"
  type        = string
}

variable "subnetwork" {
  description = "Subnetwork name to attach the cluster to"
  type        = string
}

variable "pods_range_name" {
  description = "Secondary IP range name for pods"
  type        = string
}

variable "services_range_name" {
  description = "Secondary IP range name for services"
  type        = string
}

variable "node_machine_type" {
  description = "GCE machine type for GKE nodes"
  type        = string
  default     = "e2-standard-2"
}

variable "min_node_count" {
  description = "Minimum number of nodes per zone for autoscaling"
  type        = number
  default     = 1
}

variable "max_node_count" {
  description = "Maximum number of nodes per zone for autoscaling"
  type        = number
  default     = 3
}
