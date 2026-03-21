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

variable "cart_api_image" {
  description = "Docker image for shopping cart API"
  type        = string
}

variable "db_host" {
  type = string
}

variable "db_port" {
  type = number
}

variable "db_name" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "dynamodb_table" {
  description = "DynamoDB table name for shopping carts"
  type        = string
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

# ==================== CloudWatch Log Group ====================
resource "aws_cloudwatch_log_group" "cart_api" {
  name              = "/ecs/${var.project_name}-cart-api"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-cart-api-logs"
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

  # Allow all outbound (needed for ECR pull, RDS access, DynamoDB API calls)
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

# ==================== Cart API Task Definition ====================
resource "aws_ecs_task_definition" "cart_api" {
  family                   = "${var.project_name}-cart-api"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.lab_role_arn
  task_role_arn            = var.lab_role_arn

  container_definitions = jsonencode([
    {
      name      = "${var.project_name}-cart-api"
      image     = var.cart_api_image
      essential = true

      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        # MySQL connection info (used by cart-api-mysql)
        { name = "DB_HOST", value = var.db_host },
        { name = "DB_PORT", value = tostring(var.db_port) },
        { name = "DB_NAME", value = var.db_name },
        { name = "DB_USER", value = "admin" },
        { name = "DB_PASSWORD", value = var.db_password },
        # DynamoDB info (used by cart-api-dynamodb)
        { name = "DYNAMODB_TABLE", value = var.dynamodb_table },
        { name = "AWS_REGION", value = var.aws_region }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.cart_api.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "cart-api"
        }
      }
    }
  ])
}

# ==================== Cart API ECS Service ====================
resource "aws_ecs_service" "cart_api" {
  name            = "${var.project_name}-cart-api-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.cart_api.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = "${var.project_name}-cart-api"
    container_port   = 8080
  }

  depends_on = [aws_ecs_task_definition.cart_api]
}

# ==================== Outputs ====================
output "cluster_name" {
  value = aws_ecs_cluster.main.name
}

output "cart_api_service_name" {
  value = aws_ecs_service.cart_api.name
}

output "ecs_security_group_id" {
  value = aws_security_group.ecs_tasks.id
}
