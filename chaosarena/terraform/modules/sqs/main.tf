resource "aws_sqs_queue" "photos" {
  name                       = "${var.service_name}-photo-processing"
  # Worker uses 20s long polling; visibility timeout should be longer
  # than the max processing time to avoid duplicate processing
  visibility_timeout_seconds = 300
  message_retention_seconds  = 86400 # 1 day
}
