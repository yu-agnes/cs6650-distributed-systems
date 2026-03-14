variable "project_name" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "sns_topic_arn" {
  type = string
}

variable "lab_role_arn" {
  type = string
}

variable "lambda_zip_path" {
  description = "Path to the Lambda deployment zip file"
  type        = string
}

# ==================== Lambda Function ====================
resource "aws_lambda_function" "order_processor" {
  function_name = "${var.project_name}-order-processor"
  role          = var.lab_role_arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  memory_size   = 512
  timeout       = 30

  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)

  tags = {
    Name = "${var.project_name}-lambda-processor"
  }
}

# ==================== SNS Trigger ====================
# Allow SNS to invoke this Lambda function
resource "aws_lambda_permission" "sns_invoke" {
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.order_processor.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = var.sns_topic_arn
}

# Subscribe Lambda to SNS topic
resource "aws_sns_topic_subscription" "lambda_subscription" {
  topic_arn = var.sns_topic_arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.order_processor.arn
}

# ==================== CloudWatch Log Group ====================
resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.project_name}-order-processor"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-lambda-logs"
  }
}

# ==================== Outputs ====================
output "function_name" {
  value = aws_lambda_function.order_processor.function_name
}

output "function_arn" {
  value = aws_lambda_function.order_processor.arn
}
