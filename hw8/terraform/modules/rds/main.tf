variable "project_name" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "private_subnet_ids" {
  type = list(string)
}

variable "ecs_security_group_id" {
  description = "Security group of ECS tasks - only these can access RDS"
  type        = string
}

variable "db_password" {
  description = "MySQL root password"
  type        = string
  sensitive   = true
}

# ==================== DB Subnet Group ====================
# RDS requires a subnet group spanning at least 2 AZs
resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${var.project_name}-db-subnet-group"
  }
}

# ==================== Security Group for RDS ====================
# Only allow MySQL traffic (port 3306) from ECS tasks
resource "aws_security_group" "rds" {
  name        = "${var.project_name}-rds-sg"
  description = "Allow MySQL access from ECS tasks only"
  vpc_id      = var.vpc_id

  ingress {
    description     = "MySQL from ECS"
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [var.ecs_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-rds-sg"
  }
}

# ==================== RDS MySQL Instance ====================
resource "aws_db_instance" "mysql" {
  identifier     = "${var.project_name}-mysql"
  engine         = "mysql"
  engine_version = "8.0"
  instance_class = "db.t3.micro"

  allocated_storage = 20
  storage_type      = "gp2"

  db_name  = "shopping"
  username = "admin"
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Assignment requirements: skip snapshot, disable deletion protection
  skip_final_snapshot    = true
  deletion_protection    = false

  # Disable multi-AZ for free tier
  multi_az = false

  tags = {
    Name = "${var.project_name}-mysql"
  }
}

# ==================== Outputs ====================
output "db_endpoint" {
  description = "RDS endpoint (host:port)"
  value       = aws_db_instance.mysql.endpoint
}

output "db_host" {
  description = "RDS hostname only"
  value       = aws_db_instance.mysql.address
}

output "db_port" {
  value = aws_db_instance.mysql.port
}

output "db_name" {
  value = aws_db_instance.mysql.db_name
}

output "rds_security_group_id" {
  value = aws_security_group.rds.id
}
