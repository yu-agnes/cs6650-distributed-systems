variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "hw7"
}

variable "num_workers" {
  description = "Number of worker goroutines in the processor (Phase 5: try 1, 5, 20, 100)"
  type        = number
  default     = 1
}
