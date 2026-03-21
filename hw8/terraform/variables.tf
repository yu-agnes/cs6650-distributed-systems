variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "hw8"
}

variable "db_password" {
  description = "MySQL database password"
  type        = string
  sensitive   = true
}

variable "lab_role_arn" {
  description = "ARN of the LabRole IAM role"
  type        = string
  default     = "arn:aws:iam::801832435422:role/LabRole"
}
