resource "aws_iam_role" "lambda_role" {
  name = "blackabbot_lambda_role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "blackabbot_lambda" {
  function_name = "blackab-telegram-bot"
  filename      = "../build/function.zip"
  role          = aws_iam_role.lambda_role.arn
  handler       = "./run"

  # The filebase64sha256() function is available in Terraform 0.11.12 and later
  # For Terraform 0.11.11 and earlier, use the base64sha256() function and the file() function:
  # source_code_hash = "${base64sha256(file("lambda_function_payload.zip"))}"
  source_code_hash = filebase64sha256("../build/function.zip")

  runtime = "go1.x"

  environment {
    variables = {
      TELEGRAM_BOT_TOKEN = var.telegram_bot_token
    }
  }
}
