locals {
  function_name = "blackabbot-webhook"
}

data "aws_caller_identity" "current" {}

locals {
  prefix        = "git"
  account_id    = data.aws_caller_identity.current.account_id
  ecr_image_tag = "latest"
}

resource "aws_ecr_repository" "this" {
  name = local.function_name
}



resource "aws_iam_role" "lambda_role" {
  name = "blackabbot-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid : "AssumeRole"
      Action : "sts:AssumeRole",
      Effect : "Allow",
      Principal : {
        Service : "lambda.amazonaws.com"
    } }]
  })
}

resource "aws_iam_policy" "lambda_policy" {
  name = "blackabbot-lambda-policy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid : "S3"
        Action : ["s3:PutObject", "s3:PutObjectAcl"],
        Effect : "Allow",
        Resource : "arn:aws:s3:::${aws_s3_bucket.audio_bucket.bucket}/*"
        }, {
        Sid : "Polly",
        Action : ["polly:*"],
        Effect : "Allow",
        Resource : "*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_attach" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}

resource "aws_lambda_function" "blackabbot_lambda" {
  function_name = local.function_name
  package_type  = "Image"
  role          = aws_iam_role.lambda_role.arn
  timeout       = 15
  image_uri     = "${aws_ecr_repository.this.repository_url}:${local.ecr_image_tag}"

  environment {
    variables = {
      TELEGRAM_BOT_TOKEN = var.telegram_bot_token
      TELEGRAM_BOT_NAME  = var.telegram_bot_name
      AUDIO_BUCKET       = aws_s3_bucket.audio_bucket.bucket
    }
  }
}

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

resource "null_resource" "initial_image" {
  depends_on = [aws_ecr_repository.this]

  provisioner "local-exec" {
    command     = <<EOF
                aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin ${local.account_id}.dkr.ecr.eu-west-1.amazonaws.com
                ECR_REPOSITORY=${aws_ecr_repository.this.repository_url}
                ECR_TAG=${local.ecr_image_tag}
                make build
                make push
        EOF
    working_dir = "${path.module}/.."
  }
}
