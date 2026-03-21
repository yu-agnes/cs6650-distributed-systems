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

# ==================== RDS MySQL ====================
module "rds" {
  source                = "./modules/rds"
  project_name          = var.project_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs.ecs_security_group_id
  db_password           = var.db_password
}

# ==================== DynamoDB ====================
module "dynamodb" {
  source       = "./modules/dynamodb"
  project_name = var.project_name
}

# ==================== ECS ====================
module "ecs" {
  source                = "./modules/ecs"
  project_name          = var.project_name
  aws_region            = var.aws_region
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  alb_security_group_id = module.alb.alb_security_group_id
  target_group_arn      = module.alb.target_group_arn
  cart_api_image        = "${module.ecr.repository_url}:latest"
  db_host               = module.rds.db_host
  db_port               = module.rds.db_port
  db_name               = module.rds.db_name
  db_password           = var.db_password
  dynamodb_table        = module.dynamodb.table_name
  lab_role_arn          = var.lab_role_arn
}
