# Distributed Web Scraping Pipeline

A distributed web scraping system built for CS6650 (Building Scalable Distributed Systems). The system processes URLs in parallel using AWS ECS Fargate workers, with SQS for job queuing and DynamoDB for result storage.

## Why I Built This

Single-threaded web scrapers are bottlenecked by network I/O latency. Each HTTP request blocks while waiting for a response, leaving compute resources idle. This project demonstrates how a queue-based distributed architecture solves this problem by allowing multiple workers to process URLs in parallel, with built-in fault tolerance and burst handling.

## Architecture

```
Client --POST /jobs--> ALB --> API Service --SendMessage--> SQS
                                  |                          |
                              PutItem/GetItem          ReceiveMessage
                                  |                          |
                               DynamoDB              Worker (1-8 tasks)
                                  |                          |
                              (job status              GET /page-N
                               + results)                    |
                                                      Target Service
```

**Services:**
- **API Service** (Go + Gin): Accepts job submissions, returns job ID immediately, enqueues to SQS
- **Worker Service** (Go): Polls SQS, scrapes URLs, stores results in DynamoDB
- **Target Service** (Go): Simulated website returning random HTML pages with 50-500ms latency

**AWS Resources** (managed by Terraform, 9 modules, 41 resources):
VPC, ALB, ECS Fargate, ECR, SQS (with Dead Letter Queue), DynamoDB

## Prerequisites

- AWS CLI configured with credentials
- Terraform >= 1.0
- Docker
- Go 1.21+
- Python 3.10+ (for Locust load testing)

## Quick Start

### 1. Deploy Infrastructure

```bash
cd terraform
terraform init
terraform apply -var="worker_count=4"
```

Note the outputs: `alb_dns_name` and ECR repo URLs.

### 2. Build and Push Docker Images

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Build and push each service
cd api
docker build --platform linux/amd64 -t <api_repo_url>:latest .
docker push <api_repo_url>:latest

cd ../worker
docker build --platform linux/amd64 -t <worker_repo_url>:latest .
docker push <worker_repo_url>:latest

cd ../target-service
docker build --platform linux/amd64 -t <target_repo_url>:latest .
docker push <target_repo_url>:latest
```

### 3. Start ECS Services

```bash
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-api-service --force-new-deployment
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-target-service --force-new-deployment
aws ecs update-service --cluster scrape-pipeline-cluster --service scrape-pipeline-worker-service --force-new-deployment
```

Wait 2-3 minutes, then verify:

```bash
curl http://<ALB_DNS>/health
# Should return: {"status":"ok"}
```

### 4. Submit a Job

```bash
# Submit URLs for scraping
curl -X POST http://<ALB_DNS>/jobs \
  -H "Content-Type: application/json" \
  -d '{"urls": ["page-1", "page-2", "page-3"]}'

# Check job status and results
curl http://<ALB_DNS>/jobs/<JOB_ID>
```

### 5. Run Load Tests

```bash
cd locust
pip install -r requirements.txt
locust -f locustfile_exp1.py --host http://<ALB_DNS>
# Open http://localhost:8089 to start the test
```

### 6. Tear Down

```bash
cd terraform
terraform destroy
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /jobs | Submit URLs for scraping. Body: `{"urls": ["page-1", ...]}` |
| GET | /jobs/:id | Get job status and results |
| GET | /health | Health check |

## Project Structure

```
final/
├── api/                    # API service (Go + Gin)
├── worker/                 # Worker service (Go)
├── target-service/         # Simulated target website (Go)
├── terraform/              # Infrastructure as Code (9 modules)
├── locust/                 # Load testing scripts (3 experiments)
└── README.md
```
