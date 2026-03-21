variable "project_name" {
  type = string
}

# ==================== DynamoDB Table ====================
# Unlike RDS MySQL, no subnet/security group needed!
# DynamoDB is a fully managed service accessed via AWS API,
# not through a network connection like MySQL port 3306.
resource "aws_dynamodb_table" "shopping_carts" {
  name         = "${var.project_name}-shopping-carts"
  billing_mode = "PAY_PER_REQUEST" # On-demand: pay per read/write, no provisioning

  # Only define the key - other attributes are schema-less
  hash_key = "cart_id" # Partition key

  attribute {
    name = "cart_id"
    type = "S" # String type
  }

  tags = {
    Name = "${var.project_name}-shopping-carts"
  }
}

# ==================== Outputs ====================
output "table_name" {
  value = aws_dynamodb_table.shopping_carts.name
}

output "table_arn" {
  value = aws_dynamodb_table.shopping_carts.arn
}
