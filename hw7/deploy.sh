#!/bin/bash
# HW7 Deployment Script
# Run this AFTER terraform apply to build and push Docker images

set -e

echo "=========================================="
echo "HW7 Deployment Script"
echo "=========================================="

# Get ECR URLs from Terraform output
RECEIVER_REPO=$(terraform -chdir=terraform output -raw receiver_ecr_url)
PROCESSOR_REPO=$(terraform -chdir=terraform output -raw processor_ecr_url)
REGION=$(terraform -chdir=terraform output -raw sns_topic_arn | cut -d: -f4)
ACCOUNT_ID=$(terraform -chdir=terraform output -raw sns_topic_arn | cut -d: -f5)

echo "Receiver repo: $RECEIVER_REPO"
echo "Processor repo: $PROCESSOR_REPO"
echo "Region: $REGION"
echo "Account ID: $ACCOUNT_ID"

# Login to ECR
echo ""
echo "Step 1: Logging in to ECR..."
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com

# Build and push receiver
echo ""
echo "Step 2: Building and pushing receiver image..."
cd src/receiver
docker build --platform linux/amd64 -t $RECEIVER_REPO:latest .
docker push $RECEIVER_REPO:latest
cd ../..

# Build and push processor
echo ""
echo "Step 3: Building and pushing processor image..."
cd src/processor
docker build --platform linux/amd64 -t $PROCESSOR_REPO:latest .
docker push $PROCESSOR_REPO:latest
cd ../..

# Force ECS to pull new images
echo ""
echo "Step 4: Updating ECS services..."
CLUSTER=$(terraform -chdir=terraform output -raw ecs_cluster_name)
aws ecs update-service --cluster $CLUSTER --service hw7-receiver-service --force-new-deployment --region $REGION > /dev/null
aws ecs update-service --cluster $CLUSTER --service hw7-processor-service --force-new-deployment --region $REGION > /dev/null

echo ""
echo "=========================================="
echo "Deployment complete!"
echo "ALB URL: http://$(terraform -chdir=terraform output -raw alb_dns_name)"
echo "Wait 2-3 minutes for ECS tasks to start, then test with:"
echo "  curl http://$(terraform -chdir=terraform output -raw alb_dns_name)/health"
echo "=========================================="
