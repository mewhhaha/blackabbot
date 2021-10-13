resource "aws_ecr_repository" "reply" {
  name = "reply"
}

data "aws_ecr_authorization_token" "reply" {
  registry_id = aws_ecr_repository.reply.registry_id
}

resource "null_resource" "reply_image" {
  depends_on = [
    data.aws_ecr_authorization_token.reply
  ]

  triggers = {
    always_run = "${timestamp()}"
  }

  provisioner "local-exec" {
    command = <<EOF
          docker login \
                  -u ${data.aws_ecr_authorization_token.reply.user_name} \
                  -p ${data.aws_ecr_authorization_token.reply.password} \
                  ${aws_ecr_repository.reply.repository_url}
          docker tag ${var.reply_image_id} ${aws_ecr_repository.reply.repository_url}:latest
          docker push ${aws_ecr_repository.reply.repository_url}:latest
    EOF
  }
}

data "aws_ecr_image" "registry_reply_image" {
  depends_on = [
    null_resource.reply_image
  ]
  repository_name = aws_ecr_repository.reply.name
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
        Action : ["s3:GetObject", "s3:PutObject", "s3:PutObjectAcl"],
        Effect : "Allow",
        Resource : "arn:aws:s3:::${aws_s3_bucket.audio_bucket.bucket}/*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "reply_lambda_attach" {
  role       = aws_iam_role.reply_lambda_role.name
  policy_arn = aws_iam_policy.reply_lambda_policy.arn
}


resource "aws_lambda_permission" "allow_bucket" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.reply_lambda.arn
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.audio_bucket.arn
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.audio_bucket.bucket

  lambda_function {
    lambda_function_arn = aws_lambda_function.reply_lambda.arn
    events              = ["s3:ObjectCreated:*"]
  }
}

resource "aws_lambda_function" "reply_lambda" {
  depends_on = [
    null_resource.reply_image
  ]

  function_name    = "reply_lambda"
  role             = aws_iam_role.reply_lambda_role.arn
  package_type     = "Image"
  source_code_hash = data.aws_ecr_image.registry_reply_image.id
  image_uri        = "${aws_ecr_repository.reply.repository_url}:latest"
  timeout          = 120

  environment {
    variables = {
      TELEGRAM_BOT_TOKEN = var.telegram_bot_token
      AUDIO_BUCKET       = aws_s3_bucket.audio_bucket.bucket
    }
  }
}

