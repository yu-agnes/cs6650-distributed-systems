# HW5 - Product API with Terraform Deployment

A RESTful Product API built with Go and Gin framework, deployed to AWS ECS/ECR using Terraform.

## Project Structure

```
hw5/
├── src/                    # Server code
│   ├── main.go             # Go server with Gin framework
│   ├── go.mod              # Go module dependencies
│   ├── go.sum              # Dependency checksums
│   └── Dockerfile          # Docker configuration
├── terraform/              # Infrastructure as Code
│   ├── main.tf             # Main Terraform configuration
│   ├── provider.tf         # AWS and Docker providers
│   ├── variables.tf        # Configurable variables
│   ├── outputs.tf          # Output values
│   └── modules/            # Terraform modules
│       ├── ecr/            # ECR repository
│       ├── ecs/            # ECS cluster and service
│       ├── logging/        # CloudWatch logs
│       └── network/        # VPC and security groups
├── locust/                 # Load testing
│   ├── locustfile_http.py  # HttpUser test
│   └── locustfile_fast.py  # FastHttpUser test
├── api.yaml                # OpenAPI specification (reference)
└── README.md               # This file
```

## Quick Start

### Option 1: Run Locally with Go

```bash
cd hw5/src

# Install dependencies
go mod tidy

# Run the server
go run main.go
```

Server starts at `http://localhost:8080`

### Option 2: Run with Docker

```bash
cd hw5/src

# Build image
docker build -t product-api .

# Run container
docker run -p 8080:8080 product-api
```

### Option 3: Deploy to AWS with Terraform

See [Deployment Instructions](#deployment-instructions) below.

---

## API Documentation

### Endpoints

| Method | Endpoint | Description | Response Codes |
|--------|----------|-------------|----------------|
| GET | `/health` | Health check | 200 |
| GET | `/products/{id}` | Get product by ID | 200, 400, 404 |
| POST | `/products/{id}/details` | Create/update product | 204, 400 |

### Product Schema

All fields are **required**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| product_id | integer | >= 1, must match URL | Unique identifier |
| sku | string | 1-100 characters | Stock Keeping Unit |
| manufacturer | string | 1-200 characters | Manufacturer name |
| category_id | integer | >= 1 | Category identifier |
| weight | integer | >= 0 | Weight in grams |
| some_other_id | integer | >= 1 | Additional identifier |

### Error Response Schema

```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable message",
  "details": "Additional details (optional)"
}
```

---

## API Examples

### 1. Health Check (200 OK)

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{"status":"healthy"}
```

### 2. Create Product (204 No Content)

```bash
curl -X POST http://localhost:8080/products/123/details \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 123,
    "sku": "ABC-123-XYZ",
    "manufacturer": "Acme Corporation",
    "category_id": 456,
    "weight": 1250,
    "some_other_id": 789
  }'
```

**Response:** No body (HTTP 204)

### 3. Get Product (200 OK)

```bash
curl http://localhost:8080/products/123
```

**Response:**
```json
{
  "product_id": 123,
  "sku": "ABC-123-XYZ",
  "manufacturer": "Acme Corporation",
  "category_id": 456,
  "weight": 1250,
  "some_other_id": 789
}
```

### 4. Product Not Found (404)

```bash
curl http://localhost:8080/products/99999
```

**Response:**
```json
{
  "error": "NOT_FOUND",
  "message": "Product not found",
  "details": "No product exists with ID 99999"
}
```

### 5. Invalid Product ID (400) - Non-numeric

```bash
curl http://localhost:8080/products/abc
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Invalid product ID",
  "details": "productId must be a positive integer"
}
```

### 6. Invalid Product ID (400) - Zero or Negative

```bash
curl http://localhost:8080/products/0
curl http://localhost:8080/products/-1
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Invalid product ID",
  "details": "productId must be a positive integer"
}
```

### 7. Product ID Mismatch (400)

```bash
curl -X POST http://localhost:8080/products/123/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":999,"sku":"ABC","manufacturer":"Test","category_id":1,"weight":0,"some_other_id":1}'
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Validation failed",
  "details": "product_id in body must match productId in path"
}
```

### 8. Empty SKU (400)

```bash
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"sku":"","manufacturer":"Test","category_id":1,"weight":0,"some_other_id":1}'
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Validation failed",
  "details": "sku must be between 1 and 100 characters"
}
```

### 9. Negative Weight (400)

```bash
curl -X POST http://localhost:8080/products/2/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":2,"sku":"TEST","manufacturer":"Test","category_id":1,"weight":-1,"some_other_id":1}'
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Validation failed",
  "details": "weight must be at least 0"
}
```

### 10. Invalid JSON (400)

```bash
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d 'not valid json'
```

**Response:**
```json
{
  "error": "INVALID_INPUT",
  "message": "Invalid JSON body",
  "details": "invalid character 'o' in literal null (expecting 'u')"
}
```

---

## Deployment Instructions

### Prerequisites

- AWS CLI installed and configured
- Terraform installed
- Docker installed and running

### Step 1: Configure AWS Credentials

Get credentials from AWS Academy Learner Lab, then:

```bash
aws configure
# Enter Access Key ID, Secret Access Key, Region (us-west-2)

aws configure set aws_session_token <YOUR-SESSION-TOKEN>
```

Verify configuration:
```bash
aws sts get-caller-identity
```

### Step 2: Deploy with Terraform

```bash
cd hw5/terraform

# Initialize Terraform
terraform init

# Deploy infrastructure
terraform apply -auto-approve
```

This will:
1. Create an ECR repository
2. Build and push the Docker image
3. Create an ECS cluster with Fargate
4. Configure networking and security groups
5. Set up CloudWatch logging

### Step 3: Get Public IP

```bash
aws ec2 describe-network-interfaces \
--network-interface-ids $(
    aws ecs describe-tasks \
    --cluster $(terraform output -raw ecs_cluster_name) \
    --tasks $(
        aws ecs list-tasks \
        --cluster $(terraform output -raw ecs_cluster_name) \
        --service-name $(terraform output -raw ecs_service_name) \
        --query 'taskArns[0]' --output text
    ) \
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
    --output text
) \
--query 'NetworkInterfaces[0].Association.PublicIp' \
--output text
```

### Step 4: Test the API

```bash
curl http://<PUBLIC-IP>:8080/health
curl http://<PUBLIC-IP>:8080/products/123
```

### Step 5: Clean Up (Important!)

```bash
terraform destroy -auto-approve
```

---

## Load Testing with Locust

### Setup

```bash
pip install locust
cd hw5/locust
```

### Run HttpUser Test

```bash
locust -f locustfile_http.py
```

### Run FastHttpUser Test

```bash
locust -f locustfile_fast.py
```

Open http://localhost:8089 and configure:
- Number of users: 100
- Spawn rate: 10
- Host: http://<YOUR-AWS-IP>:8080

### Test Results

| Metric | HttpUser | FastHttpUser |
|--------|----------|--------------|
| RPS | 48.1 | 47.7 |
| Avg Response | 91.46 ms | 89.19 ms |
| Median | 88 ms | 87 ms |
| 99%ile | 200 ms | 190 ms |
| Failures | 0% | 0% |

**Analysis:** The difference between HttpUser and FastHttpUser is minimal because the bottleneck is the server response time (~90ms), not the client. FastHttpUser's advantages become apparent only when simulating 1000+ users with very fast server responses (<10ms).

---

## Design Questions

### 1. How would you design a scalable backend for the full API?

The api.yaml defines multiple services: Products, Shopping Cart, Warehouse, and Payments. A scalable design would use:

- **Microservices Architecture**: Each service (Products, Cart, Warehouse, Payments) as independent services
- **Database per Service**: Products → PostgreSQL, Cart → Redis, Warehouse → PostgreSQL with inventory tracking
- **API Gateway**: Route requests to appropriate microservices
- **Load Balancer**: Distribute traffic across multiple instances
- **Auto-scaling**: ECS/Kubernetes with auto-scaling based on CPU/memory

### 2. What does "Terraform is a declarative language" mean?

**Declarative (Terraform):**
- You describe *what* you want (the desired end state)
- Terraform figures out *how* to achieve it
```hcl
resource "aws_ecs_cluster" "this" {
  name = "my-cluster"  # I want a cluster named "my-cluster"
}
```

**Imperative (e.g., AWS CLI scripts):**
- You describe *how* to do it (step-by-step instructions)
```bash
aws ecs create-cluster --cluster-name my-cluster
aws ecs describe-clusters --clusters my-cluster
# Handle errors, check if exists, etc.
```

**Benefits of Declarative:**
- Idempotent: Running multiple times produces the same result
- State management: Terraform tracks what exists and what needs to change
- Easier to understand: Code shows the desired architecture
- Change detection: Terraform calculates the diff and applies only necessary changes

