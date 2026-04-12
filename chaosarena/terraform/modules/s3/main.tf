resource "aws_s3_bucket" "photos" {
  bucket = var.bucket_name
}

# Disable the default "block all public access" so we can add a public policy
resource "aws_s3_bucket_public_access_block" "photos" {
  bucket = aws_s3_bucket.photos.id

  block_public_acls       = false
  ignore_public_acls      = false
  block_public_policy     = false
  restrict_public_buckets = false
}

# Allow anyone to GET objects — ChaosArena fetches photo URLs directly
resource "aws_s3_bucket_policy" "public_read" {
  bucket = aws_s3_bucket.photos.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = "*"
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.photos.arn}/*"
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.photos]
}
