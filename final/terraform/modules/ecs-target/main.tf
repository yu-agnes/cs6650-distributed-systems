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

variable "target_image" {
  type = string
}

variable "lab_role_arn" {
  type = string
}

# ==================== Security Group for Internal ALB ====================
resource "aws_security_group" "target_alb" {
  name        = "${var.project_name}-target-alb-sg"
  description = "Allow HTTP from within VPC to target ALB"
  vpc_id      = var.vpc_id

  ingress {
    description = "HTTP from VPC"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-target-alb-sg"
  }
}

# ==================== Internal ALB ====================
resource "aws_lb" "target" {
  name               = "${var.project_name}-target-alb"
  internal           = true
  load_balancer_type = "application"
  security_groups    = [aws_security_group.target_alb.id]
  subnets            = var.private_subnet_ids

  tags = {
    Name = "${var.project_name}-target-alb"
  }
}

# ==================== Target Group ====================
resource "aws_lb_target_group" "target" {
  name        = "${var.project_name}-target-tg"
  port        = 8081
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/health"
    port                = "traffic-port"
    timeout             = 5
    unhealthy_threshold = 3
  }

  tags = {
    Name = "${var.project_name}-target-tg"
  }
}

# ==================== Listener ====================
resource "aws_lb_listener" "target" {
  load_balancer_arn = aws_lb.target.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.target.arn
  }
}

# ==================== Security Group for Target ECS Tasks ====================
resource "aws_security_group" "target" {
  name        = "${var.project_name}-target-sg"
  description = "Allow traffic from internal ALB to target service"
  vpc_id      = var.vpc_id

  ingress {
    description     = "HTTP from internal ALB"
    from_port       = 8081
    to_port         = 8081
    protocol        = "tcp"
    security_groups = [aws_security_group.target_alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-target-sg"
  }
}

# ==================== CloudWatch Log Group ====================
resource "aws_cloudwatch_log_group" "target" {
  name              = "/ecs/${var.project_name}-target"
  retention_in_days = 7
}

# ==================== Task Definition ====================
resource "aws_ecs_task_definition" "target" {
  family                   = "${var.project_name}-target"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.lab_role_arn
  task_role_arn            = var.lab_role_arn

  container_definitions = jsonencode([
    {
      name      = "${var.project_name}-target"
      image     = var.target_image
      essential = true

      portMappings = [
        {
          containerPort = 8081
          hostPort      = 8081
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "PORT", value = "8081" },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.target.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "target"
        }
      }
    }
  ])
}

# ==================== ECS Service ====================
resource "aws_ecs_service" "target" {
  name            = "${var.project_name}-target-service"
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.target.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.target.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.target.arn
    container_name   = "${var.project_name}-target"
    container_port   = 8081
  }

  depends_on = [aws_lb_listener.target]
}

# ==================== Outputs ====================
output "target_service_url" {
  description = "Internal ALB DNS - workers use this to reach target service"
  value       = "http://${aws_lb.target.dns_name}"
}

output "target_security_group_id" {
  value = aws_security_group.target.id
}
