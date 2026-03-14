variable "project_name" {
  type = string
}

variable "sns_topic_arn" {
  type = string
}

# ==================== SQS Queue ====================
resource "aws_sqs_queue" "order_queue" {
  name                       = "${var.project_name}-order-processing-queue"
  visibility_timeout_seconds = 30    # default, matches assignment spec
  message_retention_seconds  = 345600 # 4 days, matches assignment spec
  receive_wait_time_seconds  = 20    # long polling, matches assignment spec

  tags = {
    Name = "${var.project_name}-order-queue"
  }
}

# ==================== Allow SNS to send messages to SQS ====================
resource "aws_sqs_queue_policy" "allow_sns" {
  queue_url = aws_sqs_queue.order_queue.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "sns.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.order_queue.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = var.sns_topic_arn
          }
        }
      }
    ]
  })
}

# ==================== Subscribe SQS to SNS ====================
# This is the key connection: messages published to SNS
# are automatically forwarded to this SQS queue
resource "aws_sns_topic_subscription" "sqs_subscription" {
  topic_arn = var.sns_topic_arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.order_queue.arn
}

# ==================== Outputs ====================
output "queue_url" {
  value = aws_sqs_queue.order_queue.id
}

output "queue_arn" {
  value = aws_sqs_queue.order_queue.arn
}
