

resource "null_resource" "reply_image" {
  depends_on = [
    null_resource.docker_login
  ]

  triggers = {
    always_run = "${timestamp()}"
  }

  provisioner "local-exec" {
    command = <<EOF
          docker tag ${var.reply_image_id} ${aws_ecr_repository.blackabbot.repository_url}/${var.reply_image_id}:latest
          docker push ${aws_ecr_repository.blackabbot.repository_url}:latest
    EOF
  }
}

data "aws_ecr_image" "registry_reply_image" {
  depends_on = [
    null_resource.reply_image
  ]
  repository_name = aws_ecr_repository.blackabbot.name
  image_tag       = "latest"
}

resource "aws_iam_role" "reply_lambda_role" {
  name = "reply-lambda-role"

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

resource "aws_iam_policy" "reply_lambda_policy" {
  name = "reply-lambda-policy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid : "S3"
        Action : ["s3:PutObject", "s3:PutObjectAcl"],
        Effect : "Allow",
        Resource : "arn:aws:s3:::${aws_s3_bucket.audio_bucket.bucket}/*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "webhook_lambda_attach" {
  role       = aws_iam_role.reply_lambda_role.name
  policy_arn = aws_iam_policy.reply_lambda_policy.arn
}

resource "aws_lambda_function" "reply_lambda" {
  depends_on = [
    null_resource.reply_image
  ]

  function_name    = "reply_lambda"
  role             = aws_iam_role.reply_lambda_role.arn
  package_type     = "Image"
  source_code_hash = data.aws_ecr_image.registry_reply_image.id
  image_uri        = "${aws_ecr_repository.blackabbot.repository_url}/${var.reply_image_id}:latest"
  timeout          = 120

  environment {
    variables = {
      TELEGRAM_BOT_TOKEN = var.telegram_bot_token
      AUDIO_BUCKET       = aws_s3_bucket.audio_bucket.bucket
    }
  }
}

