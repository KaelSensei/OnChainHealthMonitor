output "network_name" {
  description = "Name of the VPC network"
  value       = google_compute_network.vpc.name
}

output "network_id" {
  description = "ID of the VPC network"
  value       = google_compute_network.vpc.id
}

output "subnetwork_name" {
  description = "Name of the GKE subnetwork"
  value       = google_compute_subnetwork.gke_subnet.name
}

output "subnetwork_id" {
  description = "ID of the GKE subnetwork"
  value       = google_compute_subnetwork.gke_subnet.id
}

output "pods_range_name" {
  description = "Secondary IP range name for pods"
  value       = "${var.environment}-pods"
}

output "services_range_name" {
  description = "Secondary IP range name for services"
  value       = "${var.environment}-services"
}
