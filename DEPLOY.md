# EduWallet — Google Cloud + Cloudflare Deployment Guide

## Overview

This guide documents the end-to-end process for deploying the **EduWallet backend** (Go) to **Google Cloud Platform (GCP)** using the following services:

| Service | Purpose |
|---|---|
| **Cloud Run** | Containerized API server + worker (serverless) |
| **Cloud SQL (PostgreSQL 16)** | Primary database |
| **Memorystore (Redis 7)** | Caching, sessions, rate limiting |
| **Secret Manager** | JWT secrets, Razorpay keys, Resend API key |
| **Artifact Registry** | Docker image storage |
| **Cloudflare** | DNS management + CDN/proxy |

---

## Prerequisites

- A Google Cloud project with billing enabled
- `gcloud` CLI installed and authenticated (`gcloud auth login`, `gcloud config set project PROJECT_ID`)
- A Cloudflare account with your domain already added
- `docker` installed

---

## Table of Contents

1. [Local Preparations](#1-local-preparations)
2. [GCP Project Setup](#2-gcp-project-setup)
3. [Cloud SQL (PostgreSQL)](#3-cloud-sql-postgresql)
4. [Memorystore (Redis)](#4-memorystore-redis)
5. [Secret Manager](#5-secret-manager)
6. [Artifact Registry & Docker Image](#6-artifact-registry--docker-image)
7. [Cloud Run — API Service](#7-cloud-run--api-service)
8. [Cloud Run — Worker Service](#8-cloud-run--worker-service)
9. [Run Database Migrations](#9-run-database-migrations)
10. [Cloudflare DNS Setup](#10-cloudflare-dns-setup)
11. [Cloud Run Custom Domain Mapping](#11-cloud-run-custom-domain-mapping)
12. [Production Hardening Checklist](#12-production-hardening-checklist)
13. [CI/CD (Optional)](#13-cicd-optional)

---

## 1. Local Preparations

### 1.1 Enable Required GCP APIs

```bash
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  vpcaccess.googleapis.com
```

### 1.2 Set Environment Variables

```bash
export GCP_PROJECT="your-project-id"
export GCP_REGION="us-central1"
export SERVICE_NAME="eduwallet-api"
export WORKER_NAME="eduwallet-worker"
export DB_INSTANCE="eduwallet-db"
export REDIS_INSTANCE="eduwallet-redis"
export ARTIFACT_REPO="eduwallet"
export IMAGE_TAG="v1.0.0"
```

---

## 2. GCP Project Setup

### 2.1 Create a Serverless VPC Access Connector

Cloud Run needs a VPC connector to reach Cloud SQL (private IP) and Memorystore.

```bash
gcloud compute networks vpc-access connectors create serverless-vpc-conn \
  --region=$GCP_REGION \
  --range="10.8.0.0/28" \
  --network="default"
```

---

## 3. Cloud SQL (PostgreSQL)

### 3.1 Create the Instance

Use a **private IP** for security. Cloud SQL will auto-negotiate a private IP range via Private Service Access.

```bash
gcloud sql instances create $DB_INSTANCE \
  --database-version=POSTGRES_16 \
  --tier=db-f1-micro \
  --region=$GCP_REGION \
  --storage-type=SSD \
  --storage-size=10GB \
  --no-assign-ip \
  --network=default \
  --availability-type=ZONAL \
  --backup-start-time=03:00
```

> **Production:** Use `db-custom-1-3840` (1 vCPU, 3.75 GB RAM) or higher. Enable HA (`--availability-type=REGIONAL`).

### 3.2 Create the Database

```bash
gcloud sql databases create eduwallet_db --instance=$DB_INSTANCE
```

### 3.3 Create the Database User

```bash
gcloud sql users create eduwallet \
  --instance=$DB_INSTANCE \
  --password=$(openssl rand -base64 24)
```

Save the generated password. You will need it for the environment variables.

### 3.4 Get the Private IP

```bash
gcloud sql instances describe $DB_INSTANCE \
  --format="value(ipAddresses[0].ipAddress)"
```

Note this IP — it will be `DB_HOST` in production.

### 3.5 Require SSL (Production)

```bash
gcloud sql instances patch $DB_INSTANCE --require-ssl
```

Download the server CA certificate:

```bash
gcloud sql instances describe $DB_INSTANCE \
  --format="value(serverCaCert.cert)" > server-ca.pem
```

---

## 4. Memorystore (Redis)

### 4.1 Create the Redis Instance

```bash
gcloud redis instances create $REDIS_INSTANCE \
  --size=1 \
  --region=$GCP_REGION \
  --tier=STANDARD_HA \
  --redis-version=redis_7_x \
  --network=default \
  --connect-mode=PRIVATE_SERVICE_ACCESS
```

> **Development:** Use `--tier=BASIC --size=1`. **Production:** Use `STANDARD_HA` with at least 2 GB.

### 4.2 Get the Redis Connection Details

```bash
gcloud redis instances describe $REDIS_INSTANCE --region=$GCP_REGION \
  --format="value(host,port)"
```

Note the host and port. These will be `REDIS_HOST` and `REDIS_PORT` in production.

---

## 5. Secret Manager

Store all sensitive values in Secret Manager, then reference them in Cloud Run.

### 5.1 Create Secrets

```bash
# JWT Secrets (generate fresh ones)
echo -n "$(openssl rand -base64 48)" | gcloud secrets create JWT_ACCESS_SECRET --data-file=-
echo -n "$(openssl rand -base64 48)" | gcloud secrets create JWT_REFRESH_SECRET --data-file=-

# Database
echo -n "your-db-password" | gcloud secrets create DB_PASSWORD --data-file=-

# Razorpay
echo -n "rzp_live_xxx" | gcloud secrets create RAZORPAY_KEY_ID --data-file=-
echo -n "your-razorpay-secret" | gcloud secrets create RAZORPAY_KEY_SECRET --data-file=-
echo -n "your-webhook-secret" | gcloud secrets create RAZORPAY_WEBHOOK_SECRET --data-file=-

# Resend (email)
echo -n "re_xxx" | gcloud secrets create RESEND_API_KEY --data-file=-
```

### 5.2 Grant Cloud Run Access to Secrets

The Cloud Run service account needs access to read secrets:

```bash
PROJECT_NUMBER=$(gcloud projects describe $GCP_PROJECT --format="value(projectNumber)")
CLOUD_RUN_SA="$PROJECT_NUMBER-compute@developer.gserviceaccount.com"

gcloud secrets add-iam-policy-binding JWT_ACCESS_SECRET \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding JWT_REFRESH_SECRET \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding DB_PASSWORD \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding RAZORPAY_KEY_ID \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding RAZORPAY_KEY_SECRET \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding RAZORPAY_WEBHOOK_SECRET \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
gcloud secrets add-iam-policy-binding RESEND_API_KEY \
  --member="serviceAccount:$CLOUD_RUN_SA" --role="roles/secretmanager.secretAccessor"
```

---

## 6. Artifact Registry & Docker Image

### 6.1 Create the Docker Repository

```bash
gcloud artifacts repositories create $ARTIFACT_REPO \
  --repository-format=docker \
  --location=$GCP_REGION \
  --description="EduWallet Docker images"
```

### 6.2 Build & Push the Image

```bash
docker build \
  --platform=linux/amd64 \
  -t $GCP_REGION-docker.pkg.dev/$GCP_PROJECT/$ARTIFACT_REPO/$SERVICE_NAME:$IMAGE_TAG \
  .

docker push $GCP_REGION-docker.pkg.dev/$GCP_PROJECT/$ARTIFACT_REPO/$SERVICE_NAME:$IMAGE_TAG
```

The `--platform=linux/amd64` is important if you are building on an Apple Silicon Mac.

---

## 7. Cloud Run — API Service

### 7.1 Deploy the API Service

```bash
gcloud run deploy $SERVICE_NAME \
  --image=$GCP_REGION-docker.pkg.dev/$GCP_PROJECT/$ARTIFACT_REPO/$SERVICE_NAME:$IMAGE_TAG \
  --region=$GCP_REGION \
  --platform=managed \
  --port=8080 \
  --cpu=1 \
  --memory=512Mi \
  --min-instances=0 \
  --max-instances=5 \
  --concurrency=80 \
  --timeout=30s \
  --vpc-connector=serverless-vpc-conn \
  --vpc-egress=all-traffic \
  --allow-unauthenticated \
  --set-env-vars="APP_ENV=production,APP_MODE=api,APP_PORT=8080,APP_NAME=eduwallet,APP_EXTERNAL_URL=https://api.yourdomain.com,CORS_ALLOWED_ORIGINS=https://yourdomain.com,DB_HOST=10.x.x.x,DB_PORT=5432,DB_USER=eduwallet,DB_NAME=eduwallet_db,DB_SSL_MODE=require,REDIS_HOST=10.y.y.y,REDIS_PORT=6379,REDIS_DB=0,RESEND_FROM_EMAIL=noreply@yourdomain.com,RESEND_FROM_NAME=EduWallet,PAYMENT_PROVIDER=razorpay,RAZORPAY_BASE_URL=https://api.razorpay.com/v1" \
  --set-secrets="JWT_ACCESS_SECRET=JWT_ACCESS_SECRET:latest,JWT_REFRESH_SECRET=JWT_REFRESH_SECRET:latest,DB_PASSWORD=DB_PASSWORD:latest,RAZORPAY_KEY_ID=RAZORPAY_KEY_ID:latest,RAZORPAY_KEY_SECRET=RAZORPAY_KEY_SECRET:latest,RAZORPAY_WEBHOOK_SECRET=RAZORPAY_WEBHOOK_SECRET:latest,RESEND_API_KEY=RESEND_API_KEY:latest"
```

> Replace `DB_HOST` and `REDIS_HOST` with the **private IPs** from steps 3.4 and 4.2.

### 7.2 Environment Variable Reference

| Variable | Where to Get It |
|---|---|
| `APP_EXTERNAL_URL` | Your custom domain after Cloudflare setup (e.g., `https://api.yourdomain.com`) |
| `CORS_ALLOWED_ORIGINS` | Your frontend URL (comma-separated) |
| `DB_HOST` | Cloud SQL **private IP** (step 3.4) |
| `DB_SSL_MODE` | `require` — the app enforces this in production |
| `REDIS_HOST` | Memorystore **private IP** (step 4.2) |
| `PAYMENT_PROVIDER` | Must be `razorpay` in production |
| `AUTH_PUBLIC_REGISTRATION_ENABLED` | `false` by default (set to `true` only if needed) |

### 7.3 Verify the Deployment

```bash
curl -s https://api.yourdomain.com/health | python3 -m json.tool
```

Expected output:

```json
{
  "status": "ok",
  "postgres": "ok",
  "redis": "ok"
}
```

---

## 8. Cloud Run — Worker Service

The worker handles background tasks (reminder emails, etc.). Deploy it as a separate Cloud Run service.

### 8.1 Deploy the Worker

```bash
gcloud run deploy $WORKER_NAME \
  --image=$GCP_REGION-docker.pkg.dev/$GCP_PROJECT/$ARTIFACT_REPO/$SERVICE_NAME:$IMAGE_TAG \
  --region=$GCP_REGION \
  --platform=managed \
  --cpu=1 \
  --memory=256Mi \
  --min-instances=1 \
  --max-instances=1 \
  --concurrency=1 \
  --timeout=300s \
  --vpc-connector=serverless-vpc-conn \
  --vpc-egress=all-traffic \
  --no-allow-unauthenticated \
  --set-env-vars="APP_ENV=production,APP_MODE=worker,APP_PORT=8080,APP_NAME=eduwallet,WORKER_POLL_INTERVAL=5s,DB_HOST=10.x.x.x,DB_PORT=5432,DB_USER=eduwallet,DB_NAME=eduwallet_db,DB_SSL_MODE=require,REDIS_HOST=10.y.y.y,REDIS_PORT=6379,REDIS_DB=0,RESEND_FROM_EMAIL=noreply@yourdomain.com,RESEND_FROM_NAME=EduWallet,PAYMENT_PROVIDER=razorpay,RAZORPAY_BASE_URL=https://api.razorpay.com/v1" \
  --set-secrets="JWT_ACCESS_SECRET=JWT_ACCESS_SECRET:latest,JWT_REFRESH_SECRET=JWT_REFRESH_SECRET:latest,DB_PASSWORD=DB_PASSWORD:latest,RAZORPAY_KEY_ID=RAZORPAY_KEY_ID:latest,RAZORPAY_KEY_SECRET=RAZORPAY_KEY_SECRET:latest,RAZORPAY_WEBHOOK_SECRET=RAZORPAY_WEBHOOK_SECRET:latest,RESEND_API_KEY=RESEND_API_KEY:latest"
```

Key differences from the API service:
- `APP_MODE=worker` — runs the worker loop instead of the HTTP server
- `min-instances=1` — always running (cold starts are not acceptable for scheduled tasks)
- `max-instances=1` — single instance to avoid duplicate processing
- `concurrency=1` — serial processing
- `--no-allow-unauthenticated` — worker does not need public access
- `timeout=300s` — longer timeout for batch processing

---

## 9. Run Database Migrations

The project uses `golang-migrate` migrations. You need to run them against the Cloud SQL instance.

### 9.1 Option A: Run via Cloud SQL Proxy (Recommended)

```bash
# Download and start the proxy
curl -o cloud-sql-proxy https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.15.0/cloud-sql-proxy.darwin.amd64
chmod +x cloud-sql-proxy
./cloud-sql-proxy $GCP_PROJECT:$GCP_REGION:$DB_INSTANCE --port 5433 &
```

Then run migrations locally:

```bash
DB_HOST=localhost DB_PORT=5433 DB_USER=eduwallet DB_PASSWORD="your-password" DB_NAME=eduwallet_db DB_SSL_MODE=disable make migrate-up
```

### 9.2 Option B: Run via Cloud Run Job

Create a one-off Cloud Run job that runs the migrations:

```bash
gcloud run jobs create eduwallet-migrate \
  --image=$GCP_REGION-docker.pkg.dev/$GCP_PROJECT/$ARTIFACT_REPO/$SERVICE_NAME:$IMAGE_TAG \
  --region=$GCP_REGION \
  --vpc-connector=serverless-vpc-conn \
  --vpc-egress=all-traffic \
  --command="migrate" \
  --args="-path","./migrations","-database","postgres://eduwallet:PASSWORD@DB_HOST:5432/eduwallet_db?sslmode=require","up" \
  --set-env-vars="..." \
  --set-secrets="..."

gcloud run jobs execute eduwallet-migrate --region=$GCP_REGION
```

> **Note:** The current Dockerfile does not include the `golang-migrate` binary. You will need to either add it to the Dockerfile or use Option A. See the appendix for an updated Dockerfile.

---

## 10. Cloudflare DNS Setup

### 10.1 Add Your Domain to Cloudflare

If you haven't already:
1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Add your domain
3. Follow the wizard to move your domain's nameservers to Cloudflare

### 10.2 Create DNS Records

You need two records pointing to Cloud Run:

#### For the API (e.g., `api.yourdomain.com`):

Create a **CNAME** record:

| Field | Value |
|---|---|
| Type | `CNAME` |
| Name | `api` |
| Target | `ghs.googlehosted.com` |
| Proxy status | **Proxied** (orange cloud) |

#### For the root domain (if frontend is also on Cloud Run):

Create a **CNAME** record:

| Field | Value |
|---|---|
| Type | `CNAME` |
| Name | `@` |
| Target | `ghs.googlehosted.com` |
| Proxy status | **Proxied** (orange cloud) |

> **Important:** The target `ghs.googlehosted.com` is the standard Google endpoint. You can also use the Cloud Run generated URL (e.g., `eduwallet-api-xxxxx-uc.a.run.app`) as a CNAME target, but domain mapping (step 11) handles this automatically.

### 10.3 SSL/TLS Configuration

In Cloudflare, go to **SSL/TLS** → **Overview**:

- **Mode:** `Full (strict)` — Cloudflare validates the Google-managed certificate.

> If you use `Flexible` mode, Cloudflare will serve its own certificate but the connection to Cloud Run will be HTTP (not ideal). `Full (strict)` ensures end-to-end HTTPS.

---

## 11. Cloud Run Custom Domain Mapping

### 11.1 Important Note on Domain Mapping

**Cloud Run domain mappings are in Preview** and not recommended for production due to latency issues. Google recommends using a **global external Application Load Balancer** for production.

Two options:

#### Option A: Cloud Run Domain Mapping (Simpler, Preview)

```bash
# Verify domain ownership
gcloud domains verify yourdomain.com

# Map the domain
gcloud beta run domain-mappings create \
  --service=$SERVICE_NAME \
  --domain=api.yourdomain.com \
  --region=$GCP_REGION
```

Get the DNS records to add to Cloudflare:

```bash
gcloud beta run domain-mappings describe --domain=api.yourdomain.com
```

This will output `A`, `AAAA`, and `CNAME` records. Add these to Cloudflare instead of the generic `ghs.googlehosted.com`.

> **Cloudflare note:** If you have Cloudflare proxy enabled, the Google-managed certificate issuance may fail because Cloudflare intercepts the validation request. Temporarily disable the proxy (gray cloud) during initial setup, then re-enable after the certificate is issued.

#### Option B: Global External Application Load Balancer (Recommended for Production)

This is the production-grade approach. Steps:

1. Create a **serverless NEG** pointing to your Cloud Run service
2. Create a **managed SSL certificate**
3. Create a **global external HTTP(S) load balancer**
4. Point Cloudflare to the load balancer's IP

```bash
# 1. Create serverless NEG
gcloud compute network-endpoint-groups create eduwallet-neg \
  --region=$GCP_REGION \
  --network-endpoint-type=serverless \
  --cloud-run-service=$SERVICE_NAME

# 2. Create backend service
gcloud compute backend-services create eduwallet-backend \
  --load-balancing-scheme=EXTERNAL \
  --global

gcloud compute backend-services add-backend eduwallet-backend \
  --global \
  --network-endpoint-group=eduwallet-neg \
  --network-endpoint-group-region=$GCP_REGION

# 3. Create URL map
gcloud compute url-maps create eduwallet-url-map \
  --default-service=eduwallet-backend

# 4. Create managed SSL certificate
gcloud compute ssl-certificates create eduwallet-cert \
  --domains=api.yourdomain.com \
  --global

# 5. Create HTTPS proxy
gcloud compute target-https-proxies create eduwallet-https-proxy \
  --url-map=eduwallet-url-map \
  --ssl-certificates=eduwallet-cert

# 6. Create forwarding rule (reserve a static IP first)
gcloud compute addresses create eduwallet-lb-ip --global
LB_IP=$(gcloud compute addresses describe eduwallet-lb-ip --global --format="value(address)")

gcloud compute forwarding-rules create eduwallet-https-rule \
  --load-balancing-scheme=EXTERNAL \
  --network-tier=PREMIUM \
  --address=$LB_IP \
  --global \
  --target-https-proxy=eduwallet-https-proxy \
  --ports=443
```

After this, create an **A record** in Cloudflare pointing to the load balancer's static IP (`$LB_IP`), with proxy enabled (orange cloud).

---

## 12. Production Hardening Checklist

The app enforces these validations in `config.go` at startup. The service will fail to start if any of these are missing.

### Required

- [ ] `APP_ENV=production`
- [ ] `APP_EXTERNAL_URL` is a valid `https://` URL (not localhost)
- [ ] `CORS_ALLOWED_ORIGINS` is set (at least one `https://` origin, no `*`)
- [ ] `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` are **different** and each ≥ 32 chars
- [ ] `DB_SSL_MODE` is not `disable` (use `require`)
- [ ] `RESEND_API_KEY` is set
- [ ] `RESEND_FROM_EMAIL` is a real address (not `example.com`)
- [ ] `PAYMENT_PROVIDER=razorpay`
- [ ] `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET` are set (not placeholders)

### Recommended

- [ ] Cloud SQL: Enable automated backups, set retention to 7+ days
- [ ] Cloud SQL: Enable high availability (`REGIONAL`) for production
- [ ] Memorystore: Use `STANDARD_HA` tier
- [ ] Secret Manager: Rotate JWT secrets every 90 days
- [ ] Cloud Run: Set `min-instances=1` to avoid cold starts on the API
- [ ] Cloud Run: Configure [Cloud Armor](https://cloud.google.com/armor) for DDoS protection
- [ ] Enable [Cloud Audit Logs](https://cloud.google.com/logging/docs/audit) for all services
- [ ] Set up billing alerts to avoid surprise costs

---

## 13. CI/CD (Optional)

### 13.1 Cloud Build — Push to Deploy

Create a `cloudbuild.yaml` in the repository root:

```yaml
steps:
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '--platform=linux/amd64'
      - '-t'
      - '${_REGION}-docker.pkg.dev/$PROJECT_ID/${_REPO}/${_SERVICE}:$SHORT_SHA'
      - '.'

  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - '${_REGION}-docker.pkg.dev/$PROJECT_ID/${_REPO}/${_SERVICE}:$SHORT_SHA'

  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: 'gcloud'
    args:
      - 'run'
      - 'deploy'
      - '${_SERVICE}'
      - '--image=${_REGION}-docker.pkg.dev/$PROJECT_ID/${_REPO}/${_SERVICE}:$SHORT_SHA'
      - '--region=${_REGION}'
      - '--platform=managed'

substitutions:
  _SERVICE: 'eduwallet-api'
  _REGION: 'us-central1'
  _REPO: 'eduwallet'

images:
  - '${_REGION}-docker.pkg.dev/$PROJECT_ID/${_REPO}/${_SERVICE}:$SHORT_SHA'
```

### 13.2 Connect Cloud Build to Your Repo

```bash
gcloud builds triggers create github \
  --name="eduwallet-deploy" \
  --repo-owner="your-username" \
  --repo-name="eduwallet-backend" \
  --branch-pattern="^main$" \
  --build-config="cloudbuild.yaml"
```

---

## Appendix A: Dockerfile with Migrate Binary

If you want to run migrations from within the container (for Cloud Run Jobs), add the `golang-migrate` CLI to the build stage:

```dockerfile
# ── Stage 1: Build ────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api

# Install golang-migrate CLI
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/migrations ./migrations

EXPOSE 8080

CMD ["./api"]
```

---

## Appendix B: Quick Cost Estimate

| Resource | Tier | Monthly (approx.) |
|---|---|---|
| Cloud Run API | 1 vCPU, 512 MB, 0-5 instances | $0–$30 (pay-per-request) |
| Cloud Run Worker | 1 vCPU, 256 MB, 1 instance | ~$15 |
| Cloud SQL | db-f1-micro, 10 GB SSD | ~$10 |
| Memorystore | Basic, 1 GB | ~$30 |
| Secret Manager | 7 secrets | < $1 |
| Artifact Registry | < 1 GB storage | < $1 |
| **Total** | | **~$60–$90/month** |

> Production tiers (Cloud SQL Custom, HA, Memorystore STANDARD_HA) will increase costs to ~$150–$250/month.

---

## Appendix C: Troubleshooting

### Cloud Run fails to start / health check fails

Check logs:
```bash
gcloud run services logs read $SERVICE_NAME --region=$GCP_REGION
```

Common causes:
- `JWT_ACCESS_SECRET must be at least 32 characters` — check your Secret Manager values
- `DB_SSL_MODE=disable is not allowed in production` — set `DB_SSL_MODE=require`
- `APP_EXTERNAL_URL is required in production` — must be set in production
- Database connection refused — check VPC connector and private IP
- `PAYMENT_PROVIDER must be razorpay in production` — set `PAYMENT_PROVIDER=razorpay`

### Cannot connect to Cloud SQL

1. Verify the VPC connector is in the same region
2. Verify the Cloud SQL instance has a private IP assigned
3. Verify `DB_HOST` is the private IP
4. Try connecting via Cloud SQL Proxy from your local machine to verify credentials

### Domain mapping not working

1. Verify DNS records are propagated: `dig api.yourdomain.com`
2. If using Cloudflare proxy, temporarily disable it (gray cloud) during certificate issuance
3. Verify domain ownership is confirmed in Search Console
4. Check the domain mapping status: `gcloud beta run domain-mappings describe --domain=api.yourdomain.com`

### Cloudflare SSL issues with Cloud Run

If you see SSL errors after enabling Cloudflare proxy:
1. Go to Cloudflare → SSL/TLS → **Edge Certificates**
2. Disable "Always Use HTTPS" temporarily
3. Wait for Google-managed certificate to issue (up to 24 hours)
4. Re-enable "Always Use HTTPS"
5. Set SSL/TLS mode to **Full (strict)**