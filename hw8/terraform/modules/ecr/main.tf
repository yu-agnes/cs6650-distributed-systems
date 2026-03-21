variable "project_name" {
  type = string
}

resource "aws_ecr_repository" "cart_api" {
  name         = "${var.project_name}-cart-api"
  force_delete = true

  tags = {
    Name = "${var.project_name}-cart-api"
  }
}

output "repository_url" {
  value = aws_ecr_repository.cart_api.repository_url
}
