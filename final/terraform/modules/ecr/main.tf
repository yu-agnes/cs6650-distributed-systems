variable "project_name" {
  type = string
}

resource "aws_ecr_repository" "api" {
  name         = "${var.project_name}-api"
  force_delete = true

  tags = {
    Name = "${var.project_name}-api"
  }
}

resource "aws_ecr_repository" "worker" {
  name         = "${var.project_name}-worker"
  force_delete = true

  tags = {
    Name = "${var.project_name}-worker"
  }
}

resource "aws_ecr_repository" "target" {
  name         = "${var.project_name}-target"
  force_delete = true

  tags = {
    Name = "${var.project_name}-target"
  }
}

output "api_repo_url" {
  value = aws_ecr_repository.api.repository_url
}

output "worker_repo_url" {
  value = aws_ecr_repository.worker.repository_url
}

output "target_repo_url" {
  value = aws_ecr_repository.target.repository_url
}
