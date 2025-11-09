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

### 🎯 Phase 2: Pattern Analysis & Diagnosis Endpoints

AURA includes advanced pattern detection to identify and diagnose system issues automatically.

#### 23. Analyze Service

Runs all pattern detectors on a service and returns diagnosis.

```bash
curl -s http://localhost:8081/api/v1/analyze/sample-app | jq .
```

**Detectors Run:**

- ✅ Memory Leak Detection (linear regression analysis)
- ✅ Deployment Bug Detection (error rate comparison)
- ✅ Cascade Failure Detection (related service impact)
- ✅ External Failure Detection (dependency issues)
- ✅ Resource Exhaustion Detection (CPU/memory thresholds)

**Response Example:**

```json
{
  "service": "sample-app",
  "diagnosis": {
    "problem_type": "DEPLOYMENT_BUG",
    "confidence": 100,
    "description": "Detected deployment bug: error rate increased significantly",
    "evidence": {
      "before_error_rate": 0,
      "after_error_rate": 15.5,
      "before_period": "5m before latest deployment",
      "after_period": "5m after latest deployment"
    },
    "all_detections": [
      {
        "type": "DEPLOYMENT_BUG",
        "confidence": 100,
        "description": "Error rate spike detected after deployment"
      }
    ],
    "multiple_problems": true,
    "high_confidence_count": 1
  }
}
```

#### 24. Analyze All Services

Runs pattern analysis on all known services.

```bash
curl -s http://localhost:8081/api/v1/analyze/all | jq .
```

#### 25. Get Diagnosis History

Retrieves past diagnoses for a service.

```bash
curl -s "http://localhost:8081/api/v1/diagnoses/sample-app?limit=10" | jq .
```

#### 26. Get All Diagnoses

Retrieves diagnoses across all services.

```bash
curl -s "http://localhost:8081/api/v1/diagnoses?limit=50" | jq .
```

---

### 🔬 Phase 2.5: Advanced Analysis Endpoints

Enhanced statistical analysis and correlation features.

#### 27. Pattern Analysis

Detect trends, change points, and behavioral patterns in metrics.

```bash
curl -s "http://localhost:8081/api/v1/pattern/sample-app/memory_usage?duration=1h" | jq .
```

**Features:**

- Linear regression trend detection
- Volatility analysis (coefficient of variation)
- Change point detection (statistical significance)
- Seasonality detection (autocorrelation)

#### 28. Anomaly Detection

Multi-method anomaly detection using statistical techniques.

```bash
curl -s "http://localhost:8081/api/v1/anomaly/sample-app/error_rate?method=combined&duration=30m" | jq .
```

**Methods Available:**

- `zscore` - Z-score outlier detection (3σ threshold)
- `iqr` - Interquartile Range method (robust to outliers)
- `ema` - Exponential Moving Average deviation
- `combined` - Weighted combination of all methods

**Query Parameters:**

- `method` - Detection method (default: combined)
- `duration` - Time window (default: 30m)

#### 29. Service Correlation Analysis

Find correlated metrics and services to identify dependencies.

```bash
curl -s "http://localhost:8081/api/v1/correlations/sample-app?duration=1h&min_correlation=0.7" | jq .
```

**Features:**

- Pearson correlation coefficient calculation
- Cross-correlation with time lags
- Identification of leading/lagging indicators
- Correlation strength categorization

#### 30. Cascade Failure Risk

Assess risk of cascading failures across services.

```bash
curl -s "http://localhost:8081/api/v1/cascade-risk/sample-app?duration=1h" | jq .
```

**Analysis Includes:**

- Service dependency mapping
- Error rate correlation across services
- Cascade risk scoring (0-100)
- High-risk service identification

---

### Prometheus Metrics Export

#### 31. Get Prometheus Metrics

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
