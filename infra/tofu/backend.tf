# Remote state in S3 with DynamoDB locking.
# Bootstrap once:
#   aws s3 mb s3://ridego-tofu-state
#   aws dynamodb create-table \
#     --table-name ridego-tofu-lock \
#     --attribute-definitions AttributeName=LockID,AttributeType=S \
#     --key-schema AttributeName=LockID,KeyType=HASH \
#     --billing-mode PAY_PER_REQUEST

terraform {
  backend "s3" {
    bucket         = "ridego-tofu-state"
    key            = "ridego/${var.env}/terraform.tfstate"
    region         = "eu-west-1"
    encrypt        = true
    dynamodb_table = "ridego-tofu-lock"
  }
}
