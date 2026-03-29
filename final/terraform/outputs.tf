output "alb_dns_name" {
  description = "ALB URL - use this for Locust target"
  value       = module.alb.alb_dns_name
}

output "api_repo_url" {
  value = module.ecr.api_repo_url
}

output "worker_repo_url" {
  value = module.ecr.worker_repo_url
}

output "target_repo_url" {
  value = module.ecr.target_repo_url
}

output "sqs_queue_url" {
  value = module.sqs.queue_url
}

output "dynamodb_table_name" {
  value = module.dynamodb.table_name
}

output "cluster_name" {
  value = module.ecs_api.cluster_name
}
