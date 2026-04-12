output "queue_url" {
  value = aws_sqs_queue.photos.url
}

output "queue_arn" {
  value = aws_sqs_queue.photos.arn
}
