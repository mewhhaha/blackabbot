resource "aws_ecr_repository" "blackabbot" {
  name = "blackabbot"
}

resource "null_resource" "image" {
  depends_on = [
    aws_ecr_repository.blackabbot
  ]

  provisioner "local-exec" {
    command = <<EOF
          aws ecr get-login-password \
            --region eu-west-1 \
            | docker login \
              --username AWS \
              --password-stdin \
              ${aws_ecr_repository.blackabbot.repository_url}
          docker tag ${var.webhook_image_id} ${aws_ecr_repository.blackabbot.repository_url}:latest
          docker push ${aws_ecr_repository.blackabbot.repository_url}:latest
    EOF
  }
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
  function_name = "blackab-telegram-bot"
  role          = aws_iam_role.lambda_role.arn
  package_type  = "Image"
  # filename         = "../build/webhook/function.zip"
  # source_code_hash = filebase64sha256("../build/webhook/function.zip")
  # handler          = "./run"
  # runtime          = "go1.x"
  image_uri = "${aws_ecr_repository.blackabbot.repository_url}:latest"
  timeout   = 15


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
