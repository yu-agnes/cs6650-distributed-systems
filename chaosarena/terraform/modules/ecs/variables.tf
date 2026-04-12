variable "service_name" {
  type = string
}

variable "region" {
  type = string
}

variable "api_image" {
  type = string
}

variable "worker_image" {
  type = string
}

variable "container_port" {
  type = number
}

variable "subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

variable "execution_role_arn" {
  type = string
}

variable "task_role_arn" {
  type = string
}

variable "log_group_name" {
  type = string
}

variable "worker_log_group_name" {
  type = string
}

variable "target_group_arn" {
  type = string
}

variable "api_task_count" {
  type    = number
  default = 1
}

variable "worker_task_count" {
  type    = number
  default = 1
}

variable "env_vars" {
  type = list(object({
    name  = string
    value = string
  }))
  default = []
}
