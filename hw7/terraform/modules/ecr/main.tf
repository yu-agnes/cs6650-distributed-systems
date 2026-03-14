variable "project_name" {
  type = string
}

resource "aws_ecr_repository" "receiver" {
  name         = "${var.project_name}-receiver"
  force_delete = true

  tags = {
    Name = "${var.project_name}-receiver"
  }
}

resource "aws_ecr_repository" "processor" {
  name         = "${var.project_name}-processor"
  force_delete = true

  tags = {
    Name = "${var.project_name}-processor"
  }
}

output "receiver_repo_url" {
  value = aws_ecr_repository.receiver.repository_url
}

output "processor_repo_url" {
  value = aws_ecr_repository.processor.repository_url
}
