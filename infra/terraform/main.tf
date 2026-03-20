provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

module "networking" {
  source = "./modules/networking"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment
}

module "gke" {
  source = "./modules/gke"

  project_id          = var.project_id
  region              = var.region
  zone                = var.zone
  environment         = var.environment
  cluster_name        = var.cluster_name
  network             = module.networking.network_name
  subnetwork          = module.networking.subnetwork_name
  pods_range_name     = module.networking.pods_range_name
  services_range_name = module.networking.services_range_name
  node_machine_type   = var.node_machine_type
  min_node_count      = var.min_node_count
  max_node_count      = var.max_node_count
}
