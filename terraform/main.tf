terraform {
  backend "s3" {
    encrypt        = "true"
    bucket         = "terraform-state-storage-b0adf7e2-fcaa-4e8d-8825-358c01b89fb0"
    dynamodb_table = "terraform-state"
    key            = "blackabbot/terraform.tfstate"
    region         = "eu-west-1"
    profile        = "ci"
  }

  required_version = ">= 1.0.7"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 3.42.0"
    }
  }
}
