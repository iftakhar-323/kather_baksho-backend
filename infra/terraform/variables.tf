variable "aws_region" {
  description = "AWS deployment region"
  type        = "string"
  default     = "ap-southeast-1"
}

variable "environment" {
  description = "Target deployment environment (staging, production)"
  type        = "string"
  default     = "production"
}

variable "instance_type" {
  description = "EC2 instance sizing (t3.medium recommended for full polyglot stack)"
  type        = "string"
  default     = "t3.medium"
}

variable "vpc_cidr" {
  description = "CIDR block for the dedicated VPC"
  type        = "string"
  default     = "10.0.0.0/16"
}

variable "subnet_cidr" {
  description = "CIDR block for the public subnet"
  type        = "string"
  default     = "10.0.1.0/24"
}

variable "ssh_public_key" {
  description = "Public SSH key for EC2 administrative access"
  type        = "string"
  default     = ""
}

variable "admin_cidr" {
  description = "CIDR allowed for SSH and internal dashboard access"
  type        = "string"
  default     = "0.0.0.0/0"
}

