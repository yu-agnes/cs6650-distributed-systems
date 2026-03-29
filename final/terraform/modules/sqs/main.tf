variable "project_name" {
  type = string
}

# ==================== Main Queue ====================
resource "aws_sqs_queue" "scrape_jobs" {
  name                       = "${var.project_name}-scrape-jobs"
  visibility_timeout_seconds = 120  # 2 min - enough time for worker to scrape URLs
  message_retention_seconds  = 86400 # 1 day
  receive_wait_time_seconds  = 10    # long polling - reduces empty receives

  tags = {
    Name = "${var.project_name}-scrape-jobs"
  }
}

# DLQ will be added for Experiment 3
# resource "aws_sqs_queue" "scrape_jobs_dlq" { ... }

# ==================== Outputs ====================
output "queue_url" {
  value = aws_sqs_queue.scrape_jobs.url
}

output "queue_arn" {
  value = aws_sqs_queue.scrape_jobs.arn
}
