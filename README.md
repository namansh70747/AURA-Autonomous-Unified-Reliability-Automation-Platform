# 🚀 AURA - Autonomous Unified Reliability Automation Platform

A comprehensive platform for autonomous system reliability, anomaly detection, and automated decision-making using Kubernetes, Prometheus, and PostgreSQL.

**Current Version:** 0.1.0  
**Status:** ✅ Production Ready  
**Architecture:** Docker + Kubernetes + Prometheus

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Prerequisites](#prerequisites)
3. [Setup Instructions](#setup-instructions)
4. [API Endpoints](#api-endpoints)
5. [Testing Endpoints](#testing-endpoints)
6. [Troubleshooting](#troubleshooting)

---

## 🚀 Quick Start

### One-Command Setup (Recommended)

```bash
# Navigate to project
cd /path/to/AURA-Autonomous-Unified-Reliability-Automation-Platform

# Run complete setup
make start

# Verify everything is working
make test-endpoints
```

**That's it!** All services will be running:
- ✅ AURA API (port 8081)
- ✅ Prometheus (port 9090)
- ✅ PostgreSQL (port 5432)
- ✅ Sample App (port 8080)

---

## 📦 Prerequisites

Before starting, ensure you have:

```bash
# Check Go
go version
# Expected: go1.21+

# Check Docker
docker --version

# Check Docker Compose
docker-compose version

# Check Kubernetes
minikube status

# If minikube is not running
minikube start
```

---

## 🔧 Setup Instructions

### Step 1: Install Dependencies

```bash
make setup
```

### Step 2: Setup Kubeconfig for Docker (One-Time)

```bash
make setup-kubeconfig
```

**What this does:**
- ✅ Copies `~/.kube/config` to `~/.kube_docker/config`
- ✅ Replaces `127.0.0.1` with `host.docker.internal`
- ✅ Adds `insecure-skip-tls-verify: true` for self-signed certificates
- ✅ Automatically handles all Docker networking issues

**Verify kubeconfig:**

```bash
cat ~/.kube_docker/config | grep "server:"
```

### Step 3: Build Linux Binaries

```bash
make build-linux
```

### Step 4: Start AURA with Docker

```bash
make docker-up
```

---

## 📊 API Endpoints

### Status & Health Endpoints

#### 1. Health Check
```bash
curl -s http://localhost:8081/health | jq .
```
**Response:** `200 OK`

#### 2. Readiness Probe
```bash
curl -s http://localhost:8081/ready | jq .
```
**Response:** `200 OK`

#### 3. Service Status
```bash
curl -s http://localhost:8081/api/v1/status | jq .
```
**Response:** `200 OK`

---

### Kubernetes Endpoints

#### 4. List All Pods
```bash
curl -s http://localhost:8081/api/v1/kubernetes/pods | jq .
```

#### 5. Get Pod Details
```bash
curl -s http://localhost:8081/api/v1/kubernetes/pods/{pod-name} | jq .
```

#### 6. Get Pod Metrics
```bash
curl -s "http://localhost:8081/api/v1/kubernetes/pods/{pod-name}/metrics?duration=1h" | jq .
```

#### 7. Get Cluster Events
```bash
curl -s http://localhost:8081/api/v1/kubernetes/events | jq .
```

#### 8. Get Pod Events
```bash
curl -s "http://localhost:8081/api/v1/kubernetes/events/{pod-name}?duration=1h" | jq .
```

#### 9. Get Namespace Summary
```bash
curl -s "http://localhost:8081/api/v1/kubernetes/namespace/summary?namespace=default" | jq .
```

---

### Prometheus Endpoints

#### 10. Prometheus Health
```bash
curl -s http://localhost:8081/api/v1/prometheus/health | jq .
```

#### 11. Prometheus Targets
```bash
curl -s http://localhost:8081/api/v1/prometheus/targets | jq .
```

#### 12. Prometheus Query
```bash
curl -s "http://localhost:8081/api/v1/prometheus/query?query=cpu_usage&service=sample-app" | jq .
```

#### 13. Prometheus Metrics Summary
```bash
curl -s "http://localhost:8081/api/v1/prometheus/metrics/summary?duration=1h" | jq .
```

---

### Metrics Endpoints

#### 14. Get Service Metrics
```bash
curl -s http://localhost:8081/api/v1/metrics/sample-app | jq .
```

#### 15. Get All Services
```bash
curl -s http://localhost:8081/api/v1/metrics/services | jq .
```

#### 16. Get Metric Statistics
```bash
curl -s "http://localhost:8081/api/v1/metrics/sample-app/cpu_usage/stats" | jq .
```

#### 17. Get Metric History
```bash
curl -s "http://localhost:8081/api/v1/metrics/sample-app/history?type=cpu_usage&duration=1h" | jq .
```

---

### Decision Endpoints

#### 18. Get Recent Decisions
```bash
curl -s http://localhost:8081/api/v1/decisions | jq .
```

#### 19. Get Decision Statistics
```bash
curl -s http://localhost:8081/api/v1/decisions/stats | jq .
```

#### 20. Get Decision by ID
```bash
curl -s http://localhost:8081/api/v1/decisions/{id} | jq .
```

---

### Observer Endpoints

#### 21. Observer Health
```bash
curl -s http://localhost:8081/api/v1/observer/health | jq .
```

#### 22. Observer Metrics
```bash
curl -s "http://localhost:8081/api/v1/observer/metrics?service=sample-app" | jq .
```

---

### Prometheus Metrics Export

#### 23. Get Prometheus Metrics
```bash
curl -s http://localhost:8081/metrics | head -20
```

---

## 🧪 Testing Endpoints

### Run All Tests at Once

```bash
make test-endpoints
```

### Quick Verification Script

```bash
#!/bin/bash

echo "=== AURA PLATFORM TEST ==="
echo ""

echo "1. AURA Status"
curl -s http://localhost:8081/api/v1/status | jq . && echo ""

echo "2. Kubernetes Pods"
curl -s http://localhost:8081/api/v1/kubernetes/pods | jq . && echo ""

echo "3. Kubernetes Events"
curl -s http://localhost:8081/api/v1/kubernetes/events | jq . && echo ""

echo "4. Kubernetes Namespace Summary"
curl -s http://localhost:8081/api/v1/kubernetes/namespace/summary | jq . && echo ""

echo "5. Prometheus Health"
curl -s http://localhost:8081/api/v1/prometheus/health | jq . && echo ""

echo "6. Prometheus Targets"
curl -s http://localhost:8081/api/v1/prometheus/targets | jq . && echo ""

echo "7. Service Metrics"
curl -s http://localhost:8081/api/v1/metrics/sample-app | jq . && echo ""

echo "✅ All endpoint tests completed!"
```

---

## 🛠️ Common Commands

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f aura
docker-compose logs -f postgres
docker-compose logs -f prometheus
docker-compose logs -f sample-app
```

### Check Service Health

```bash
# All containers
docker-compose ps

# Database
docker exec aura-postgres pg_isready -U aura -d aura_db

# Kubernetes
kubectl get pods
kubectl get events
```

### Rebuild Services

```bash
# Rebuild only AURA
docker-compose up -d --build aura

# Rebuild everything
docker-compose down
docker-compose up -d
```

---

## 🛑 Stopping AURA

### Stop All Services

```bash
make docker-down
```

### Full Cleanup

```bash
make clean-all
```

---

## 🔧 Troubleshooting

### Kubernetes Endpoints Return 404

```bash
# Re-run kubeconfig setup
make setup-kubeconfig

# Restart AURA
docker-compose restart aura && sleep 5

# Test
curl -s http://localhost:8081/api/v1/kubernetes/pods | jq .
```

### Port Already in Use

```bash
# Find process
lsof -i :8081

# Kill it
kill -9 <PID>

# Restart
docker-compose down && docker-compose up -d
```

### Services Won't Start

```bash
# Clean rebuild
docker-compose down -v
docker-compose build --no-cache
docker-compose up -d
```

### Database Connection Failed

```bash
# Reset database
make db-reset

# Check connection
docker exec aura-postgres pg_isready -U aura -d aura_db
```

---

**Last Updated:** October 25, 2025  
**Version:** 0.1.0  
**Status:** ✅ Production Ready
