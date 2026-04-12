# The public URL to submit to ChaosArena as base_url
output "api_base_url" {
  description = "Submit this as base_url to ChaosArena"
  value       = "http://${module.alb.dns_name}"
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = module.alb.dns_name
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = module.sqs.queue_url
}

output "s3_bucket_name" {
  description = "S3 bucket name"
  value       = module.s3.bucket_name
}

output "ecr_api_url" {
  description = "ECR repository URL for the API image"
  value       = module.ecr.api_repository_url
}

output "ecr_worker_url" {
  description = "ECR repository URL for the Worker image"
  value       = module.ecr.worker_repository_url
}
