output "alb_dns_name" {
  description = "ALB DNS name - use this for testing"
  value       = module.alb.alb_dns_name
}

output "ecr_repo_url" {
  description = "ECR repository URL for cart-api image"
  value       = module.ecr.repository_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = module.ecs.cluster_name
}

output "rds_endpoint" {
  description = "RDS MySQL endpoint"
  value       = module.rds.db_endpoint
}

output "dynamodb_table_name" {
  description = "DynamoDB table name"
  value       = module.dynamodb.table_name
}
