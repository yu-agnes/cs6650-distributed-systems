variable "project_name" {
  type = string
}

resource "aws_sns_topic" "order_events" {
  name = "${var.project_name}-order-processing-events"

  tags = {
    Name = "${var.project_name}-order-events"
  }
}

output "topic_arn" {
  value = aws_sns_topic.order_events.arn
}
