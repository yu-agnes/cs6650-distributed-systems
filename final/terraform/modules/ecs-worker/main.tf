variable "project_name" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "private_subnet_ids" {
  type = list(string)
}

variable "cluster_id" {
  type = string
}

variable "worker_image" {
  type = string
}

variable "lab_role_arn" {
  type = string
}

variable "sqs_queue_url" {
  type = string
}

variable "dynamodb_table_name" {
  type = string
}

variable "target_service_url" {
  description = "Internal URL of the target service to scrape"
  type        = string
}

variable "desired_count" {
  description = "Number of worker tasks - change this for Experiment 1"
  type        = number
  default     = 1
}

# ==================== Security Group ====================
resource "aws_security_group" "worker" {
  name        = "${var.project_name}-worker-sg"
  description = "Worker tasks - outbound only"
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-worker-sg"
  }
}

# ==================== CloudWatch Log Group ====================
resource "aws_cloudwatch_log_group" "worker" {
  name              = "/ecs/${var.project_name}-worker"
  retention_in_days = 7
}

# ==================== Task Definition ====================
resource "aws_ecs_task_definition" "worker" {
  family                   = "${var.project_name}-worker"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.lab_role_arn
  task_role_arn            = var.lab_role_arn

  container_definitions = jsonencode([
    {
      name      = "${var.project_name}-worker"
      image     = var.worker_image
      essential = true

      environment = [
        { name = "AWS_REGION", value = var.aws_region },
        { name = "SQS_QUEUE_URL", value = var.sqs_queue_url },
        { name = "DYNAMODB_TABLE", value = var.dynamodb_table_name },
        { name = "TARGET_SERVICE_URL", value = var.target_service_url },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.worker.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "worker"
        }
      }
    }
  ])
}

# ==================== ECS Service ====================
resource "aws_ecs_service" "worker" {
  name            = "${var.project_name}-worker-service"
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.worker.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.worker.id]
    assign_public_ip = false
  }

  depends_on = [aws_ecs_task_definition.worker]
}

# ==================== Outputs ====================
output "worker_service_name" {
  value = aws_ecs_service.worker.name
}

output "worker_security_group_id" {
  value = aws_security_group.worker.id
}
