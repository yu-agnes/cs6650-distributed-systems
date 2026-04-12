variable "aws_region" {
  type    = string
  default = "us-west-2"
}

variable "service_name" {
  type    = string
  default = "album-store"
}

variable "container_port" {
  type    = number
  default = 8080
}

# How many API tasks to run (increase for load testing)
variable "api_task_count" {
  type    = number
  default = 8
}

# How many Worker tasks to run
variable "worker_task_count" {
  type    = number
  default = 5
}

variable "log_retention_days" {
  type    = number
  default = 7
}

# S3 bucket name — must be globally unique across all of AWS
variable "s3_bucket_name" {
  type    = string
  default = "cs6650-album-store-yuye-w2"
}
