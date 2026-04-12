resource "aws_ecs_cluster" "this" {
  name = "${var.service_name}-cluster"
}

# ── API Task Definition ──────────────────────────────────────────────────────
# Given S15 can send 200MB uploads, give the API task more memory.

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.service_name}-api"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  container_definitions = jsonencode([{
    name      = "${var.service_name}-api"
    image     = var.api_image
    essential = true

    portMappings = [{
      containerPort = var.container_port
      protocol      = "tcp"
    }]

    environment = var.env_vars

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = var.log_group_name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "api"
      }
    }
  }])
}

# ── API ECS Service (connected to ALB) ──────────────────────────────────────

resource "aws_ecs_service" "api" {
  name            = "${var.service_name}-api"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = var.api_task_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = var.security_group_ids
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = "${var.service_name}-api"
    container_port   = var.container_port
  }
}

# ── Worker Task Definition ───────────────────────────────────────────────────
# Worker only processes photos — no port needed, no ALB.

resource "aws_ecs_task_definition" "worker" {
  family                   = "${var.service_name}-worker"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  container_definitions = jsonencode([{
    name      = "${var.service_name}-worker"
    image     = var.worker_image
    essential = true

    # Workers don't expose any ports
    environment = [for e in var.env_vars : e if e.name != "PORT"]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = var.worker_log_group_name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "worker"
      }
    }
  }])
}

# ── Worker ECS Service (no ALB) ──────────────────────────────────────────────

resource "aws_ecs_service" "worker" {
  name            = "${var.service_name}-worker"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.worker.arn
  desired_count   = var.worker_task_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = var.security_group_ids
    assign_public_ip = true
  }
}
