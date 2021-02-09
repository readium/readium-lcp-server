#!/bin/bash
# Add S3 and IAM access resources for the LCP instance using AWS CLI utility

# Load env values
. .env

# Create S3 resource

# Create S3 bucket with public read policy
aws s3api create-bucket --bucket $READIUM_CONTENT_S3_BUCKET --region $AWS_REGION --create-bucket-configuration LocationConstraint=$AWS_REGION --output json
cp files/s3-public-read-access.json.default files/s3-public-read-access.json
sed -i "s/S3BUCKETNAME/$READIUM_CONTENT_S3_BUCKET/g" files/s3-public-read-access.json
aws s3api put-bucket-policy --bucket $READIUM_CONTENT_S3_BUCKET --policy file://files/s3-public-read-access.json --output json

# Add IAM resources

# Update write policy with proper bucket name
cp files/s3-group-access-policy.json.default files/s3-group-access-policy.json
sed -i "s/S3BUCKETNAME/$READIUM_CONTENT_S3_BUCKET/g" files/s3-group-access-policy.json

# Create IAM group and write policy resources
POLICY_ARN=$(aws iam create-policy --policy-name $AWS_S3_WRITE_POLICY --policy-document file://files/s3-group-access-policy.json --output json | jq -r '.Policy.Arn')
aws iam create-group --group-name $AWS_S3_USER_GROUP
aws iam attach-group-policy --policy-arn $POLICY_ARN --group-name $AWS_S3_USER_GROUP
# Use pound sign as a delimiter, since the ARN has a slash in it (same for secret below)
sed -i "s#POLICY_ARN=''#POLICY_ARN='$POLICY_ARN'#g" .env

# Create IAM user and assign to group
aws iam create-user --user-name $AWS_S3_USER
aws iam add-user-to-group --user-name $AWS_S3_USER --group-name $AWS_S3_USER_GROUP
KEY_RESULT=$(aws iam create-access-key --user-name $AWS_S3_USER --output json)

# Assign the values of the user key/id and secret to the proper variable in .env
USER_KEY=$(echo $KEY_RESULT | jq -r '.AccessKey.AccessKeyId')
USER_SECRET=$(echo $KEY_RESULT | jq -r '.AccessKey.SecretAccessKey')
sed -i "s/S3_KEY=''/S3_KEY='$USER_KEY'/g" .env
sed -i "s#S3_SECRET=''#S3_SECRET='$USER_SECRET'#g" .env

# Create AWS credentials file for LCP user
cp files/credentials.default files/aws_credentials
sed -i "s/KEYID/$USER_KEY/g" files/aws_credentials
sed -i "s#ACCESSKEY#$USER_SECRET#g" files/aws_credentials

# Create new Dockerfile with command to copy the credentials into the lcpserver image
cp Dockerfile Dockerfile.orig
sed -i '/READIUM_LCPSERVER_CONFIG/i COPY files/aws_credentials /root/.aws/credentials' Dockerfile