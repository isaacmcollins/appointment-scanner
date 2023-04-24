terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 3.20.0"
    }
  }
  backend "s3" {
    bucket = "collisaa"
    key    = "appointment-scanner-tf-state/tfstate"
    region = "us-west-2"
  }
  required_version = ">= 0.14"
}

provider "aws" {
  region = "us-west-2"
}

data "aws_availability_zones" "zones" {}