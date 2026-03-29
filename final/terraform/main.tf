# ==================== VPC ====================
module "vpc" {
  source       = "./modules/vpc"
  project_name = var.project_name
}

# ==================== ECR ====================
module "ecr" {
  source       = "./modules/ecr"
  project_name = var.project_name
}

# ==================== ALB ====================
module "alb" {
  source            = "./modules/alb"
  project_name      = var.project_name
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
}

# ==================== SQS ====================
module "sqs" {
  source       = "./modules/sqs"
  project_name = var.project_name
}

# ==================== DynamoDB ====================
module "dynamodb" {
  source       = "./modules/dynamodb"
  project_name = var.project_name
}

# ==================== ECS - API Service ====================
module "ecs_api" {
  source                = "./modules/ecs-api"
  project_name          = var.project_name
  aws_region            = var.aws_region
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  alb_security_group_id = module.alb.alb_security_group_id
  target_group_arn      = module.alb.target_group_arn
  api_image             = "${module.ecr.api_repo_url}:latest"
  lab_role_arn          = var.lab_role_arn
  sqs_queue_url         = module.sqs.queue_url
  dynamodb_table_name   = module.dynamodb.table_name
}

# ==================== ECS - Target Service ====================
module "ecs_target" {
  source             = "./modules/ecs-target"
  project_name       = var.project_name
  aws_region         = var.aws_region
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  cluster_id         = module.ecs_api.cluster_id
  target_image       = "${module.ecr.target_repo_url}:latest"
  lab_role_arn       = var.lab_role_arn
}

# ==================== ECS - Worker Service ====================
module "ecs_worker" {
  source              = "./modules/ecs-worker"
  project_name        = var.project_name
  aws_region          = var.aws_region
  vpc_id              = module.vpc.vpc_id
  private_subnet_ids  = module.vpc.private_subnet_ids
  cluster_id          = module.ecs_api.cluster_id
  worker_image        = "${module.ecr.worker_repo_url}:latest"
  lab_role_arn        = var.lab_role_arn
  sqs_queue_url       = module.sqs.queue_url
  dynamodb_table_name = module.dynamodb.table_name
  target_service_url  = module.ecs_target.target_service_url
  desired_count       = var.worker_count
}
