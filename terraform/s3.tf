resource "aws_s3_bucket" "audio_bucket" {
  bucket = "audio-bucket-01d19784-9eea-4632-9b70-c46c60acef3e"
  acl    = "public-read"

  lifecycle_rule {
    id      = "all"
    enabled = true

    expiration {
      days = 90
    }
  }
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.audio_bucket.bucket

  lambda_function {
    lambda_function_arn = aws_lambda_function.reply_lambda.arn
    events              = ["s3:ObjectCreated:*"]
  }
}
