resource "aws_cloudwatch_log_group" "api" {
  name              = "/ecs/${var.service_name}-api"
  retention_in_days = var.retention_in_days
}

resource "aws_cloudwatch_log_group" "worker" {
  name              = "/ecs/${var.service_name}-worker"
  retention_in_days = var.retention_in_days
}
