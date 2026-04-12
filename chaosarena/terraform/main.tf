# Reuse the existing LabRole for all ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

# ── Storage & Messaging ──────────────────────────────────────────────────────

module "dynamodb" {
  source       = "./modules/dynamodb"
  service_name = var.service_name
}

module "s3" {
  source      = "./modules/s3"
  bucket_name = var.s3_bucket_name
}

module "sqs" {
  source       = "./modules/sqs"
  service_name = var.service_name
}

# ── Container Registries ─────────────────────────────────────────────────────

module "ecr" {
  source       = "./modules/ecr"
  service_name = var.service_name
}

# Build and push the API image
resource "docker_image" "api" {
  name = "${module.ecr.api_repository_url}:latest"
  build {
    context    = "${path.module}/.."
    dockerfile = "Dockerfile.api"
  }
  depends_on = [module.ecr]
}

resource "docker_registry_image" "api" {
  name = docker_image.api.name
}

# Build and push the Worker image
resource "docker_image" "worker" {
  name = "${module.ecr.worker_repository_url}:latest"
  build {
    context    = "${path.module}/.."
    dockerfile = "Dockerfile.worker"
  }
  depends_on = [module.ecr]
}

resource "docker_registry_image" "worker" {
  name = docker_image.worker.name
}

# ── Networking ───────────────────────────────────────────────────────────────

module "network" {
  source         = "./modules/network"
  service_name   = var.service_name
  container_port = var.container_port
}

# ── Load Balancer ────────────────────────────────────────────────────────────

module "alb" {
  source         = "./modules/alb"
  service_name   = var.service_name
  vpc_id         = module.network.vpc_id
  subnet_ids     = module.network.subnet_ids
  container_port = var.container_port
}

# ── Logging ──────────────────────────────────────────────────────────────────

module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = var.log_retention_days
}

# ── ECS Cluster + Services ───────────────────────────────────────────────────

module "ecs" {
  source = "./modules/ecs"

  service_name       = var.service_name
  region             = var.aws_region
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name        = module.logging.log_group_name
  worker_log_group_name = module.logging.worker_log_group_name

  api_image          = "${module.ecr.api_repository_url}:latest"
  worker_image       = "${module.ecr.worker_repository_url}:latest"
  container_port     = var.container_port
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [module.network.security_group_id]
  target_group_arn   = module.alb.target_group_arn

  api_task_count    = var.api_task_count
  worker_task_count = var.worker_task_count

  # Environment variables injected into both containers
  env_vars = [
    { name = "AWS_REGION",    value = var.aws_region },
    { name = "ALBUMS_TABLE",  value = module.dynamodb.albums_table_name },
    { name = "PHOTOS_TABLE",  value = module.dynamodb.photos_table_name },
    { name = "S3_BUCKET",     value = module.s3.bucket_name },
    { name = "SQS_QUEUE_URL", value = module.sqs.queue_url },
    { name = "PORT",          value = tostring(var.container_port) },
  ]

  depends_on = [
    docker_registry_image.api,
    docker_registry_image.worker,
    module.alb,
  ]
}
