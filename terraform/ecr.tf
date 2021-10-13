resource "aws_ecr_repository" "blackabbot" {
  name = "blackabbot"
}

data "aws_ecr_authorization_token" "blackabbot" {
  registry_id = aws_ecr_repository.blackabbot.registry_id
}


resource "null_resource" "docker_login" {
  depends_on = [
    data.aws_ecr_authorization_token.blackabbot
  ]

  triggers = {
    always_run = "${timestamp()}"
  }

  provisioner "local-exec" {
    command = <<EOF
          docker login \
            -u ${data.aws_ecr_authorization_token.blackabbot.user_name} \
            -p ${data.aws_ecr_authorization_token.blackabbot.password} \
            ${aws_ecr_repository.blackabbot.repository_url}
    EOF
  }
}
