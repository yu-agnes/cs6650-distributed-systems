output "alb_dns_name" {
  description = "ALB DNS name - use this for Locust testing"
  value       = module.alb.alb_dns_name
}

output "receiver_ecr_url" {
  description = "ECR repository URL for receiver image"
  value       = module.ecr.receiver_repo_url
}

output "processor_ecr_url" {
  description = "ECR repository URL for processor image"
  value       = module.ecr.processor_repo_url
}

output "sns_topic_arn" {
  description = "SNS topic ARN"
  value       = module.sns.topic_arn
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = module.sqs.queue_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = module.ecs.cluster_name
}
