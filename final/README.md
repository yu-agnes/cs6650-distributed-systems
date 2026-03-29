# Distributed Web Scraping Pipeline

A distributed web scraping system deployed on AWS ECS/Fargate. Users submit URLs via HTTP API, workers process them asynchronously via SQS, and results are stored in DynamoDB.

## Architecture

```
Client → ALB → API Service → SQS → Worker Service(s) → Target Service
                  ↕                       ↕
               DynamoDB              DynamoDB
```

## Prerequisites

- AWS CLI configured with Learner Lab credentials
- Terraform >= 1.0
- Docker
- Go 1.21+
- Python 3.10+ (for Locust)

## Quick Start

### 1. Deploy Infrastructure

```bash
cd terraform
terraform init
terraform apply
```

Note the outputs: `alb_dns_name`, ECR repo URLs.

### 2. Build and Push Docker Images

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Build and push each service (from project root)
# API
cd api
docker build --platform linux/amd64 -t <api_repo_url>:latest .
docker push <api_repo_url>:latest

# Worker
cd ../worker
docker build --platform linux/amd64 -t <worker_repo_url>:latest .
docker push <worker_repo_url>:latest

# Target
cd ../target-service
docker build --platform linux/amd64 -t <target_repo_url>:latest .
docker push <target_repo_url>:latest
```

### 3. Force ECS to Pull New Images

```bash
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-api-service --force-new-deployment
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-worker-service --force-new-deployment
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-target-service --force-new-deployment
```

### 4. Test

```bash
# Submit a job
curl -X POST http://<ALB_DNS>/jobs \
  -H "Content-Type: application/json" \
  -d '{"urls": ["page-1", "page-2", "page-3"]}'

# Check status
curl http://<ALB_DNS>/jobs/<JOB_ID>
```

### 5. Run Experiment 1

```bash
cd locust
pip install -r requirements.txt

# With 1 worker (default)
locust -f locustfile.py --host http://<ALB_DNS> --headless -u 1 -r 1 --run-time 5m

# Change to 2 workers
cd ../terraform
terraform apply -var="worker_count=2"
# Wait for ECS to stabilize, then re-run locust

# Repeat for 4 and 8 workers
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /jobs | Submit URLs for scraping. Body: `{"urls": ["page-1", ...]}` |
| GET | /jobs/:id | Get job status and results |
| GET | /health | Health check |

## Experiments

1. **Worker Scaling vs Throughput** - 500 URLs with 1/2/4/8 workers
2. **Burst Absorption** - 10x traffic spike with SQS as buffer
3. **Worker Failure Recovery** - Kill workers, observe SQS visibility timeout behavior
