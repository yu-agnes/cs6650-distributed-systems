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

# ==================== SNS ====================
module "sns" {
  source       = "./modules/sns"
  project_name = var.project_name
}

# ==================== SQS ====================
module "sqs" {
  source        = "./modules/sqs"
  project_name  = var.project_name
  sns_topic_arn = module.sns.topic_arn
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
  receiver_image        = "${module.ecr.receiver_repo_url}:latest"
  processor_image       = "${module.ecr.processor_repo_url}:latest"
  sns_topic_arn         = module.sns.topic_arn
  sqs_queue_url         = module.sqs.queue_url
  sqs_queue_arn         = module.sqs.queue_arn
  num_workers           = var.num_workers
  lab_role_arn          = "arn:aws:iam::801832435422:role/LabRole"
}

# ==================== Lambda (Part III) ====================
module "lambda" {
  source          = "./modules/lambda"
  project_name    = var.project_name
  aws_region      = var.aws_region
  sns_topic_arn   = module.sns.topic_arn
  lab_role_arn    = "arn:aws:iam::801832435422:role/LabRole"
  lambda_zip_path = "${path.module}/lambda.zip"
}
