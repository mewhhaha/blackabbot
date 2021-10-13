resource "null_resource" "webhook_image" {
  depends_on = [
    null_resource.docker_login
  ]

  triggers = {
    always_run = "${timestamp()}"
  }

  provisioner "local-exec" {
    command = <<EOF
          docker tag ${var.webhook_image_id} ${aws_ecr_repository.blackabbot.repository_url}/${var.webhook_image_id}:latest
          docker push ${aws_ecr_repository.blackabbot.repository_url}/${var.webhook_image_id}:latest
    EOF
  }
}

data "aws_ecr_image" "registry_webhook_image" {
  depends_on = [
    null_resource.webhook_image
  ]
  repository_name = aws_ecr_repository.blackabbot.name
  image_tag       = "latest"
}

resource "aws_iam_role" "webhook_lambda_role" {
  name = "webhook-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid : "AssumeRole",
      Action : "sts:AssumeRole",
      Effect : "Allow",
      Principal : {
        Service : "lambda.amazonaws.com"
    } }]
  })
}

resource "aws_iam_policy" "webhook_lambda_policy" {
  name = "webhook-lambda-policy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid : "Polly",
        Action : ["polly:*"],
        Effect : "Allow",
        Resource : "*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "reply_lambda_attach" {
  role       = aws_iam_role.webhook_lambda_role.name
  policy_arn = aws_iam_policy.webhook_lambda_policy.arn
}

resource "aws_lambda_function" "webhook_lambda" {
  depends_on = [
    null_resource.webhook_image
  ]

  function_name    = "webhook_lambda"
  role             = aws_iam_role.webhook_lambda_role.arn
  package_type     = "Image"
  source_code_hash = data.aws_ecr_image.registry_webhook_image.id
  image_uri        = "${aws_ecr_repository.blackabbot.repository_url}/${var.webhook_image_id}:latest"
  timeout          = 120


  environment {
    variables = {
      TELEGRAM_BOT_NAME = var.telegram_bot_name
      AUDIO_BUCKET      = aws_s3_bucket.audio_bucket.bucket
    }
  }
}

