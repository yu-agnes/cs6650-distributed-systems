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

variable "alb_security_group_id" {
  type = string
}

variable "target_group_arn" {
  type = string
}

variable "receiver_image" {
  type = string
}

variable "processor_image" {
  type = string
}

variable "sns_topic_arn" {
  type = string
}

variable "sqs_queue_url" {
  type = string
}

variable "sqs_queue_arn" {
  type = string
}

variable "num_workers" {
  type    = number
  default = 1
}

variable "lab_role_arn" {
  type = string
}

# ==================== ECS Cluster ====================
resource "aws_ecs_cluster" "main" {
  name = "${var.project_name}-cluster"

  tags = {
    Name = "${var.project_name}-cluster"
  }
}

# ==================== CloudWatch Log Groups ====================
resource "aws_cloudwatch_log_group" "receiver" {
  name              = "/ecs/${var.project_name}-receiver"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-receiver-logs"
  }
}

resource "aws_cloudwatch_log_group" "processor" {
  name              = "/ecs/${var.project_name}-processor"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-processor-logs"
  }
}

# ==================== Security Group for ECS Tasks ====================
resource "aws_security_group" "ecs_tasks" {
  name        = "${var.project_name}-ecs-tasks-sg"
  description = "Allow traffic from ALB to ECS tasks"
  vpc_id      = var.vpc_id

  # Allow traffic from ALB on port 8080
  ingress {
    description     = "HTTP from ALB"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [var.alb_security_group_id]
  }

  # Allow all outbound (needed for ECR pull, SNS/SQS API calls via NAT)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-ecs-tasks-sg"
  }
}

# ==================== Receiver Task Definition ====================
resource "aws_ecs_task_definition" "receiver" {
  family                   = "${var.project_name}-receiver"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.lab_role_arn
  task_role_arn            = var.lab_role_arn

  container_definitions = jsonencode([
    {
      name      = "${var.project_name}-receiver"
      image     = var.receiver_image
      essential = true

      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "SNS_TOPIC_ARN", value = var.sns_topic_arn },
        { name = "AWS_REGION", value = var.aws_region }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.receiver.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "receiver"
        }
      }
    }
  ])
}

# ==================== Processor Task Definition ====================
resource "aws_ecs_task_definition" "processor" {
  family                   = "${var.project_name}-processor"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.lab_role_arn
  task_role_arn            = var.lab_role_arn

  container_definitions = jsonencode([
    {
      name      = "${var.project_name}-processor"
      image     = var.processor_image
      essential = true

      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "SQS_QUEUE_URL", value = var.sqs_queue_url },
        { name = "AWS_REGION", value = var.aws_region },
        { name = "NUM_WORKERS", value = tostring(var.num_workers) }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.processor.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "processor"
        }
      }
    }
  ])
}

# ==================== Receiver ECS Service ====================
resource "aws_ecs_service" "receiver" {
  name            = "${var.project_name}-receiver-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.receiver.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = "${var.project_name}-receiver"
    container_port   = 8080
  }

  depends_on = [aws_ecs_task_definition.receiver]
}

# ==================== Processor ECS Service ====================
resource "aws_ecs_service" "processor" {
  name            = "${var.project_name}-processor-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.processor.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  depends_on = [aws_ecs_task_definition.processor]
}

# ==================== Outputs ====================
output "cluster_name" {
  value = aws_ecs_cluster.main.name
}

output "receiver_service_name" {
  value = aws_ecs_service.receiver.name
}

output "processor_service_name" {
  value = aws_ecs_service.processor.name
}
