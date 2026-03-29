variable "project_name" {
  type = string
}

# ==================== Jobs Table ====================
# Partition key: jobID
# Each item stores: jobID, status, urls, results, createdAt, completedAt
resource "aws_dynamodb_table" "jobs" {
  name         = "${var.project_name}-jobs"
  billing_mode = "PAY_PER_REQUEST" # no capacity planning needed
  hash_key     = "jobID"

  attribute {
    name = "jobID"
    type = "S"
  }

  tags = {
    Name = "${var.project_name}-jobs"
  }
}

# ==================== Outputs ====================
output "table_name" {
  value = aws_dynamodb_table.jobs.name
}

output "table_arn" {
  value = aws_dynamodb_table.jobs.arn
}
