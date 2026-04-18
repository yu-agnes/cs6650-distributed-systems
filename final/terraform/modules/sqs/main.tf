variable "project_name" {
  type = string
}

# ==================== Dead Letter Queue ====================
resource "aws_sqs_queue" "scrape_jobs_dlq" {
  name                      = "${var.project_name}-scrape-jobs-dlq"
  message_retention_seconds = 86400 # 1 day

  tags = {
    Name = "${var.project_name}-scrape-jobs-dlq"
  }
}

# ==================== Main Queue ====================
resource "aws_sqs_queue" "scrape_jobs" {
  name                       = "${var.project_name}-scrape-jobs"
  visibility_timeout_seconds = 120  # 2 min - enough time for worker to scrape URLs
  message_retention_seconds  = 86400 # 1 day
  receive_wait_time_seconds  = 10    # long polling - reduces empty receives

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.scrape_jobs_dlq.arn
    maxReceiveCount     = 3  # after 3 failed attempts, move to DLQ
  })

  tags = {
    Name = "${var.project_name}-scrape-jobs"
  }
}

# ==================== Outputs ====================
output "queue_url" {
  value = aws_sqs_queue.scrape_jobs.url
}

output "queue_arn" {
  value = aws_sqs_queue.scrape_jobs.arn
}

output "dlq_url" {
  value = aws_sqs_queue.scrape_jobs_dlq.url
}

output "dlq_arn" {
  value = aws_sqs_queue.scrape_jobs_dlq.arn
}
