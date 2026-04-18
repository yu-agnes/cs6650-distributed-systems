variable "project_name" {
  type    = string
  default = "scrape-pipeline"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "lab_role_arn" {
  type    = string
  default = "arn:aws:iam::823613469987:role/ecsTaskExecutionRole"
}

variable "worker_count" {
  description = "Number of worker tasks - change for Experiment 1 (1, 2, 4, 8)"
  type        = number
  default     = 1
}
