#!/bin/bash
# Remove IAM and S3 resources for the LCP instance using AWS CLI utility

# Load env values
. .env

# Remove IAM access resources first

# Remove user account
aws iam remove-user-from-group --user-name $AWS_S3_USER --group-name $AWS_S3_USER_GROUP
aws iam delete-access-key --access-key-id $AWS_S3_KEY --user-name $AWS_S3_USER
aws iam delete-user --user-name $AWS_S3_USER

# Remove group
aws iam detach-group-policy --group-name $AWS_S3_USER_GROUP --policy-arn $AWS_S3_WRITE_POLICY_ARN
aws iam delete-group --group-name $AWS_S3_USER_GROUP

# Remove policy
aws iam delete-policy --policy-arn $AWS_S3_WRITE_POLICY_ARN
rm files/s3-group-access-policy.json

# Reset .env variables for policy ARN and S3_USER key to blanks
sed -i $'/POLICY_ARN/c\\\nAWS_S3_WRITE_POLICY_ARN=\'\'\n' .env
sed -i $'/S3_KEY/c\\\nAWS_S3_KEY=\'\'\n' .env
sed -i $'/S3_SECRET/c\\\nAWS_S3_SECRET=\'\'\n' .env

# Remove AWS credentials file
rm files/aws_credentials
if [ -f "Dockerfile.orig" ]; then
  rm Dockerfile
  mv Dockerfile.orig Dockerfile
fi

# Then remove S3 bucket and contents (only if versioning is turned off; otherwise need additional tasks)
aws s3 rb s3://$READIUM_CONTENT_S3_BUCKET --force
rm files/s3-public-read-access.json