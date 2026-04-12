resource "aws_ecr_repository" "api" {
  name = "${var.service_name}-api"
}

resource "aws_ecr_repository" "worker" {
  name = "${var.service_name}-worker"
}
