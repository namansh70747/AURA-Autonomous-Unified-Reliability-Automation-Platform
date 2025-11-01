# 🚀 AURA Main Server (main.go) - Complete API Guide

## What is main.go?

`main.go` is the **heart of AURA** - it's the main application server that:

- Initializes all AURA components (database, observer, logger)
- Exposes REST API endpoints for monitoring and automation
- Connects to Kubernetes and Prometheus
- Listens for commands and provides real-time data
- Manages graceful shutdown

Think of it as the **command center** that coordinates everything:

```
┌──────────────────────────────────────────────────────────┐
│ AURA Main Server (main.go)                               │
├──────────────────────────────────────────────────────────┤
│                                                          │
│   Database (PostgreSQL)                                  │
│  ├─ Stores metrics, decisions, events                    │
│  └─ Provides historical data                             │
│                                                          │
│   Observer (Prometheus + Kubernetes)                     │
│  ├─ Watches Kubernetes pods                              │
│  ├─ Collects metrics from Prometheus                     │
│  └─ Real-time monitoring                                 │
│                                                          │
│    REST API Server (Port 8081)                           │
│  ├─ Health checks                                        │
│  ├─ Metrics endpoints                                    │
│  ├─ Kubernetes integration                               │
│  ├─ Decision tracking                                    │
│  └─ Prometheus integration                               │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## 🎯 Initialization Flow (What Happens at Startup)

When you start AURA, this is what happens:

```
1. Load Configuration (lines 24-31)
   └─ Reads AURA_CONFIG_PATH or defaults to configs/aura.yaml

2. Initialize Logger (lines 33-38)
   └─ Sets up structured logging (using Uber's zap)

3. Connect to Database (lines 40-46)
   └─ Establishes PostgreSQL connection
   └─ Tests connectivity

4. Create Metrics Observer (lines 53-62)
   └─ Connects to Prometheus
   └─ Prepares Kubernetes access
   └─ Sets refresh interval: 10 seconds

5. Start Monitoring Goroutine (lines 64-68)
   └─ Observer runs in background
   └─ Continuously collects metrics

6. Start Console Monitor (line 75)
   └─ Prints CPU/Memory every 10 seconds to console

7. Setup HTTP Router (lines 77-137)
   └─ Registers all API endpoints
   └─ Configures middleware

8. Start HTTP Server (lines 139-147)
   └─ Listens on :8081
   └─ Ready to accept requests!
```

---

## 📊 API Endpoints Overview

| Category | Purpose | Count |
|----------|---------|-------|
| **Health** | System checks | 2 |
| **Metrics** | Performance data | 4 |
| **Decisions** | Automation history | 3 |
| **Observer** | Monitoring status | 2 |
| **Kubernetes** | Pod & event data | 6 |
| **Prometheus** | Metrics source | 4 |
| **Total** | | **21 endpoints** |

---

## 🏥 Health & Status Endpoints

### 1️⃣ Health Check

**Endpoint:** `GET /health`

**Purpose:** Check if AURA is alive and database is responding

**Curl:**

```bash
curl -X GET http://localhost:8081/health
```

**Response (Healthy):**

```json
{
  "status": "healthy",
  "timestamp": "2025-10-29T10:30:45Z",
  "version": "1.0.0"
}
```

**Response (Unhealthy):**

```json
{
  "status": "unhealthy",
  "error": "database connection failed"
}
```

**Why AURA Needs This:**

- ✅ Kubernetes readiness probes use this
- ✅ Load balancers check this for routing decisions
- ✅ Monitoring systems alert if this fails

---

### 2️⃣ Readiness Check

**Endpoint:** `GET /ready`

**Purpose:** Verify AURA is fully initialized and ready to serve traffic

**Curl:**

```bash
curl -X GET http://localhost:8081/ready
```

**Response (Ready):**

```json
{
  "status": "ready",
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Response (Not Ready):**

```json
{
  "status": "not_ready",
  "reason": "database unavailable"
}
```

**Why AURA Needs This:**

- ✅ Kubernetes startup probes use this
- ✅ Prevents traffic before fully initialized
- ✅ Graceful startup process

---

### 3️⃣ Status & Version

**Endpoint:** `GET /api/v1/status`

**Purpose:** Get basic info about AURA service

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/status
```

**Response:**

```json
{
  "service": "AURA",
  "version": "1.0.0",
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Verify correct version is running
- ✅ Admin dashboards display this info
- ✅ Troubleshooting version mismatches

---

## 📈 Metrics Endpoints

These endpoints provide **performance data** about your services and infrastructure.

### 4️⃣ Get Current Service Metrics

**Endpoint:** `GET /api/v1/metrics/:service`

**Purpose:** Get latest CPU, Memory, HTTP requests for a service

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/metrics/sample-app
```

**Response:**

```json
{
  "service_name": "sample-app",
  "timestamp": "2025-10-29T10:30:45Z",
  "metrics": {
    "cpu_usage": 45.5,
    "memory_usage": 62.3,
    "http_requests": 1250
  }
}
```

**Why AURA Needs This:**

- ✅ Real-time performance monitoring
- ✅ Detect anomalies quickly
- ✅ Feed data into decision engine

---

### 5️⃣ Get Metric Statistics (Min/Max/Avg)

**Endpoint:** `GET /api/v1/metrics/:service/:metric/stats`

**Purpose:** Get statistical analysis of a metric over time period

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/metrics/sample-app/cpu_usage/stats"
```

**Response:**

```json
{
  "service": "sample-app",
  "metric": "cpu_usage",
  "period": "1 hour",
  "min": 20.5,
  "max": 85.3,
  "avg": 52.1,
  "p95": 78.9,
  "p99": 82.5,
  "stddev": 15.3
}
```

**Why AURA Needs This:**

- ✅ Understand metric distribution
- ✅ Detect patterns (e.g., peak hours)
- ✅ Set intelligent thresholds for alerts

---

### 6️⃣ Get Metric History

**Endpoint:** `GET /api/v1/metrics/:service/history?type=cpu_usage&duration=24h`

**Purpose:** Get historical metric data over specified duration

**Curl:**

```bash
# Last 1 hour of CPU data
curl -X GET "http://localhost:8081/api/v1/metrics/sample-app/history?type=cpu_usage&duration=1h"

# Last 24 hours of memory data
curl -X GET "http://localhost:8081/api/v1/metrics/sample-app/history?type=memory_usage&duration=24h"

# Last 7 days of HTTP requests
curl -X GET "http://localhost:8081/api/v1/metrics/sample-app/history?type=http_requests&duration=168h"
```

**Response:**

```json
{
  "service": "sample-app",
  "metric_type": "cpu_usage",
  "duration": "1h",
  "data_points": 60,
  "metrics": [
    { "timestamp": "2025-10-29T09:30:45Z", "value": 45.2 },
    { "timestamp": "2025-10-29T09:31:45Z", "value": 46.1 },
    { "timestamp": "2025-10-29T09:32:45Z", "value": 44.8 }
  ],
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Trend analysis (is CPU going up over time?)
- ✅ Capacity planning
- ✅ Historical reporting and audits

---

### 7️⃣ List All Services with Metrics

**Endpoint:** `GET /api/v1/metrics/services`

**Purpose:** Get list of all services being monitored

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/metrics/services
```

**Response:**

```json
{
  "services": [
    "sample-app",
    "database-service",
    "api-gateway",
    "cache-layer"
  ],
  "count": 4,
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Service discovery
- ✅ Admin dashboards show what's monitored
- ✅ Verify services are being tracked

---

## 🤖 Decision Endpoints

These endpoints track **automation decisions** AURA made (like scaling, restarts, etc.)

### 8️⃣ Get Recent Decisions

**Endpoint:** `GET /api/v1/decisions`

**Purpose:** Get last 20 automation decisions AURA made

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/decisions
```

**Response:**

```json
{
  "decisions": [
    {
      "id": "dec-001",
      "timestamp": "2025-10-29T10:28:15Z",
      "action": "scale-deployment",
      "target": "sample-app",
      "reason": "CPU exceeded 80%",
      "confidence": 0.95,
      "status": "executed"
    },
    {
      "id": "dec-002",
      "timestamp": "2025-10-29T10:15:30Z",
      "action": "restart-pod",
      "target": "sample-app-xyz123",
      "reason": "Memory leak detected",
      "confidence": 0.87,
      "status": "executed"
    }
  ],
  "count": 20
}
```

**Why AURA Needs This:**

- ✅ Audit trail of all automation actions
- ✅ Verify decisions were good (did scaling help?)
- ✅ Machine learning feedback loop
- ✅ Compliance and blame tracking

---

### 9️⃣ Decision Statistics

**Endpoint:** `GET /api/v1/decisions/stats`

**Purpose:** Summary of decisions made in last 24 hours

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/decisions/stats
```

**Response:**

```json
{
  "period": "24 hours",
  "total_decisions": 156,
  "executed": 150,
  "failed": 3,
  "skipped": 3,
  "actions_breakdown": {
    "scale_deployment": 45,
    "restart_pod": 67,
    "update_config": 23,
    "drain_node": 12,
    "other": 9
  },
  "confidence_stats": {
    "avg_confidence": 0.88,
    "high_confidence": 145,
    "medium_confidence": 9,
    "low_confidence": 2
  }
}
```

**Why AURA Needs This:**

- ✅ Understand automation activity
- ✅ Spot issues (too many failures?)
- ✅ Optimize decision rules
- ✅ Management reports

---

### 🔟 Get Specific Decision

**Endpoint:** `GET /api/v1/decisions/:id`

**Purpose:** Deep dive into one specific decision

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/decisions/dec-001
```

**Response:**

```json
{
  "decision": {
    "id": "dec-001",
    "timestamp": "2025-10-29T10:28:15Z",
    "action": "scale-deployment",
    "target": "sample-app",
    "reason": "CPU exceeded 80% threshold",
    "metrics_at_decision": {
      "cpu": 82.5,
      "memory": 45.3,
      "http_requests": 5000
    },
    "confidence": 0.95,
    "status": "executed",
    "result": {
      "new_replicas": 5,
      "old_replicas": 3,
      "duration": "45s"
    },
    "feedback": {
      "effectiveness": 0.92,
      "notes": "Scaling resolved the issue"
    }
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Debug why specific decision was made
- ✅ Understand decision reasoning
- ✅ Machine learning learns from this

---

## 👁️ Observer Endpoints

These endpoints show status of the **monitoring system** (Observer).

### 1️⃣1️⃣ Observer Health

**Endpoint:** `GET /api/v1/observer/health`

**Purpose:** Check if observer is running and collecting metrics

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/observer/health
```

**Response:**

```json
{
  "status": "running",
  "interval": "10s"
}
```

**Why AURA Needs This:**

- ✅ Verify monitoring is active
- ✅ If observer dies, AURA goes blind!

---

### 1️⃣2️⃣ Get Observer's Current Metrics

**Endpoint:** `GET /api/v1/observer/metrics?service=sample-app`

**Purpose:** Get raw metrics from observer for specific service

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/observer/metrics?service=sample-app"
```

**Response:**

```json
{
  "service": "sample-app",
  "metrics": {
    "cpu": 45.3,
    "memory": 62.1,
    "network_in": 1024000,
    "network_out": 512000,
    "disk_io": 45.2,
    "http_latency_p95": 125,
    "http_latency_p99": 450
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Direct access to observer data
- ✅ More detailed metrics than cached ones
- ✅ Real-time monitoring dashboards

---

## 🐳 Kubernetes Integration Endpoints

These endpoints give **Kubernetes-specific** information about pods and events.

### 1️⃣3️⃣ List All Pods

**Endpoint:** `GET /api/v1/kubernetes/pods`

**Purpose:** Get all pods in monitored namespace

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/kubernetes/pods
```

**Response:**

```json
{
  "pods": [
    {
      "name": "sample-app-5f8c9d",
      "namespace": "default",
      "phase": "Running",
      "restarts": 0,
      "age": "2d",
      "cpu": "45.2m",
      "memory": "128Mi"
    },
    {
      "name": "sample-app-7x2b1c",
      "namespace": "default",
      "phase": "Running",
      "restarts": 1,
      "age": "1d",
      "cpu": "52.1m",
      "memory": "145Mi"
    }
  ],
  "count": 2
}
```

**Why AURA Needs This:**

- ✅ Know which pods are running
- ✅ Detect failed or stuck pods
- ✅ Base data for scaling decisions

---

### 1️⃣4️⃣ Get Specific Pod Details

**Endpoint:** `GET /api/v1/kubernetes/pods/:name`

**Purpose:** Deep dive into one pod's details

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/kubernetes/pods/sample-app-5f8c9d
```

**Response:**

```json
{
  "pod": {
    "name": "sample-app-5f8c9d",
    "namespace": "default",
    "phase": "Running",
    "restarts": 0,
    "age": "2d",
    "cpu": "45.2m",
    "memory": "128Mi",
    "containers": [
      {
        "name": "app",
        "image": "sample-app:v1.0",
        "status": "running"
      }
    ],
    "node": "worker-node-1",
    "ip": "10.244.0.5"
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Debug pod issues
- ✅ Understand resource allocation
- ✅ Plan replacements

---

### 1️⃣5️⃣ Get Pod's Historical Metrics

**Endpoint:** `GET /api/v1/kubernetes/pods/:name/metrics?duration=1h`

**Purpose:** See how pod's resources changed over time

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/kubernetes/pods/sample-app-5f8c9d/metrics?duration=1h"
```

**Response:**

```json
{
  "pod": "sample-app-5f8c9d",
  "duration": "1h",
  "metrics": {
    "pod_status": [
      { "timestamp": "2025-10-29T09:30Z", "restarts": 0 },
      { "timestamp": "2025-10-29T10:30Z", "restarts": 0 }
    ],
    "pod_restarts": []
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Detect flaky pods (restart spikes)
- ✅ Understand resource trends

---

### 1️⃣6️⃣ Get Kubernetes Events

**Endpoint:** `GET /api/v1/kubernetes/events`

**Purpose:** Get all Kubernetes events (pod scheduled, pulled image, etc.)

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/kubernetes/events
```

**Response:**

```json
{
  "events": [
    {
      "type": "Normal",
      "reason": "Scheduled",
      "pod": "sample-app-5f8c9d",
      "message": "Successfully assigned sample-app-5f8c9d to worker-node-1",
      "timestamp": "2025-10-29T08:15:30Z"
    },
    {
      "type": "Normal",
      "reason": "Pulled",
      "pod": "sample-app-5f8c9d",
      "message": "Container image \"sample-app:v1.0\" already present on machine",
      "timestamp": "2025-10-29T08:15:35Z"
    },
    {
      "type": "Warning",
      "reason": "FailedScheduling",
      "pod": "sample-app-xyz123",
      "message": "0/3 nodes are available: 3 Insufficient memory",
      "timestamp": "2025-10-29T09:45:12Z"
    }
  ],
  "count": 142
}
```

**Why AURA Needs This:**

- ✅ Understand what's happening in cluster
- ✅ Detect resource constraints
- ✅ Alert on failures

---

### 1️⃣7️⃣ Get Events for Specific Pod

**Endpoint:** `GET /api/v1/kubernetes/events/:podname?duration=1h`

**Purpose:** See all events for one pod

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/kubernetes/events/sample-app-5f8c9d?duration=1h"
```

**Response:**

```json
{
  "pod": "sample-app-5f8c9d",
  "duration": "1h",
  "events": [
    {
      "type": "Normal",
      "reason": "Scheduled",
      "message": "Successfully assigned",
      "timestamp": "2025-10-29T08:15:30Z"
    },
    {
      "type": "Normal",
      "reason": "Started",
      "message": "Started container",
      "timestamp": "2025-10-29T08:15:35Z"
    }
  ],
  "count": 5
}
```

**Why AURA Needs This:**

- ✅ Troubleshoot specific pod issues
- ✅ Timeline of pod's lifecycle

---

### 1️⃣8️⃣ Get Namespace Summary

**Endpoint:** `GET /api/v1/kubernetes/namespace/summary?namespace=default`

**Purpose:** Complete overview of namespace health

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/kubernetes/namespace/summary?namespace=default"
```

**Response:**

```json
{
  "summary": {
    "namespace": "default",
    "total_pods": 5,
    "running_pods": 4,
    "pending_pods": 0,
    "failed_pods": 1,
    "total_restarts": 3,
    "recent_events": 12
  },
  "pods": [
    { "name": "sample-app-5f8c9d", "phase": "Running", "restarts": 0 },
    { "name": "sample-app-7x2b1c", "phase": "Running", "restarts": 1 },
    { "name": "database-abc123", "phase": "Running", "restarts": 2 },
    { "name": "cache-def456", "phase": "Running", "restarts": 0 },
    { "name": "worker-ghi789", "phase": "Failed", "restarts": 0 }
  ],
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Quick health check of namespace
- ✅ Spot problems at a glance
- ✅ Decision engine's main input

---

## 🔥 Prometheus Integration Endpoints

These endpoints provide **metric source** information and query capabilities.

### 1️⃣9️⃣ Prometheus Health

**Endpoint:** `GET /api/v1/prometheus/health`

**Purpose:** Check if Prometheus server is reachable

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/prometheus/health
```

**Response:**

```json
{
  "status": "healthy",
  "message": "Prometheus is reachable",
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Verify metrics source is alive
- ✅ If Prometheus dies, AURA loses visibility

---

### 2️⃣0️⃣ Get Prometheus Targets

**Endpoint:** `GET /api/v1/prometheus/targets`

**Purpose:** See what services Prometheus is scraping

**Curl:**

```bash
curl -X GET http://localhost:8081/api/v1/prometheus/targets
```

**Response:**

```json
{
  "targets": [
    {
      "name": "sample-app",
      "url": "http://sample-app:8080/metrics",
      "status": "up"
    },
    {
      "name": "database",
      "url": "http://database:5432/metrics",
      "status": "up"
    },
    {
      "name": "cache",
      "url": "http://cache:6379/metrics",
      "status": "down"
    }
  ],
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Verify all services are being monitored
- ✅ Detect scrape failures

---

### 2️⃣1️⃣ Query Prometheus

**Endpoint:** `GET /api/v1/prometheus/query?query=cpu_usage&service=sample-app`

**Purpose:** Execute custom queries against Prometheus

**Curl:**

```bash
# Query CPU usage
curl -X GET "http://localhost:8081/api/v1/prometheus/query?query=cpu_usage&service=sample-app"

# Query memory usage
curl -X GET "http://localhost:8081/api/v1/prometheus/query?query=memory_usage&service=sample-app"

# Query HTTP errors
curl -X GET "http://localhost:8081/api/v1/prometheus/query?query=http_errors&service=api-gateway"
```

**Response:**

```json
{
  "query": "cpu_usage",
  "service": "sample-app",
  "result": {
    "cpu": 45.3,
    "memory": 62.1,
    "http_requests": 5000,
    "http_latency_p95": 125
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Execute complex queries
- ✅ Developers debug issues
- ✅ Custom analytics

---

### 2️⃣2️⃣ Prometheus Metrics Summary

**Endpoint:** `GET /api/v1/prometheus/metrics/summary?duration=1h`

**Purpose:** Statistical summary of all metrics over time period

**Curl:**

```bash
curl -X GET "http://localhost:8081/api/v1/prometheus/metrics/summary?duration=1h"
```

**Response:**

```json
{
  "duration": "1h",
  "summary": {
    "sample-app": {
      "cpu_usage": {
        "min": 20.5,
        "max": 85.3,
        "avg": 52.1
      },
      "memory_usage": {
        "min": 45.2,
        "max": 78.9,
        "avg": 62.1
      },
      "http_requests": {
        "min": 1000,
        "max": 8500,
        "avg": 4250
      }
    }
  },
  "timestamp": "2025-10-29T10:30:45Z"
}
```

**Why AURA Needs This:**

- ✅ Understand metric patterns
- ✅ Set intelligent thresholds
- ✅ Capacity planning

---

## 📊 System Monitoring Metrics Endpoint

### 2️⃣3️⃣ Prometheus Metrics Scrape

**Endpoint:** `GET /metrics`

**Purpose:** AURA's own metrics (for external Prometheus to scrape)

**Curl:**

```bash
curl -X GET http://localhost:8081/metrics
```

**Response (Prometheus format):**

```
# HELP aura_decisions_total Total decisions made
# TYPE aura_decisions_total counter
aura_decisions_total 1560

# HELP aura_decision_errors Total decision errors
# TYPE aura_decision_errors counter
aura_decision_errors 15

# HELP aura_http_requests_duration_seconds HTTP request duration
# TYPE aura_http_requests_duration_seconds histogram
aura_http_requests_duration_seconds_bucket{le="0.1"} 500
```

**Why AURA Needs This:**

- ✅ Meta-monitoring (monitor the monitor!)
- ✅ External Prometheus scrapes AURA's health

---

## 🎨 Architecture Visualization

```
┌────────────────────────────────────────────────────────────┐
│                   AURA REST API Server                     │
│                    (Port 8081)                             │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Health Checks (Kubernetes Probes)                    │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ GET /health       → Is AURA alive?                   │  │
│  │ GET /ready        → Is AURA ready for traffic?       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Metrics API (Time-Series Data)                       │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ GET /api/v1/metrics/:service                         │  │
│  │ GET /api/v1/metrics/:service/:metric/stats           │  │
│  │ GET /api/v1/metrics/:service/history                 │  │
│  │ GET /api/v1/metrics/services                         │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                  │
│                   PostgreSQL Database                      │
│                   (Stores metrics)                         │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Decision API (Automation History)                    │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ GET /api/v1/decisions                                │  │
│  │ GET /api/v1/decisions/stats                          │  │
│  │ GET /api/v1/decisions/:id                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│                   PostgreSQL Database                        │
│                   (Stores decisions)                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Kubernetes API (Pod & Event Data)                    │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ GET /api/v1/kubernetes/pods                          │  │
│  │ GET /api/v1/kubernetes/pods/:name                    │  │
│  │ GET /api/v1/kubernetes/events                        │  │
│  │ GET /api/v1/kubernetes/namespace/summary             │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│                   Kubernetes API Server                      │
│                   (Live pod data)                            │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Prometheus API (Metrics Source)                      │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ GET /api/v1/prometheus/health                        │  │
│  │ GET /api/v1/prometheus/targets                       │  │
│  │ GET /api/v1/prometheus/query                         │  │
│  │ GET /api/v1/prometheus/metrics/summary               │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│                   Prometheus Server                          │
│                   (Collects metrics)                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Monitoring (GET /metrics)                            │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ AURA's own metrics for external Prometheus           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔄 Request Flow Example: Decision Making

Here's what happens when AURA needs to make a decision:

```
1. Observer collects data every 10 seconds
   curl http://localhost:8081/api/v1/kubernetes/pods
   ↓
   Gets: 4 running pods, 1 pending, CPU at 85%

2. Decision Engine analyzes metrics
   curl http://localhost:8081/api/v1/metrics/sample-app
   ↓
   Gets: CPU=85%, Memory=72%, HTTP_latency_p95=450ms

3. Confidence calculation (87% confident to scale)

4. Decision is stored
   POST to database

5. You can review it later
   curl http://localhost:8081/api/v1/decisions/dec-001
   ↓
   Returns: Full decision with reasoning, metrics, and results

6. Machine learning learns
   If result was good → increase confidence
   If result was bad → adjust rules
```

---

## 🚨 Real-World Usage Scenarios

### Scenario 1: Troubleshoot High CPU

```bash
# 1. Check overall health
curl http://localhost:8081/api/v1/kubernetes/namespace/summary

# 2. Find the problematic pod
curl http://localhost:8081/api/v1/kubernetes/pods

# 3. Deep dive into that pod
curl http://localhost:8081/api/v1/kubernetes/pods/sample-app-5f8c9d

# 4. See historical metrics
curl "http://localhost:8081/api/v1/metrics/sample-app/history?duration=1h"

# 5. View decisions made
curl http://localhost:8081/api/v1/decisions
```

### Scenario 2: Understand Automation History

```bash
# 1. See recent decisions
curl http://localhost:8081/api/v1/decisions

# 2. Get statistics
curl http://localhost:8081/api/v1/decisions/stats

# 3. Examine specific decision
curl http://localhost:8081/api/v1/decisions/dec-001
```

### Scenario 3: Setup Monitoring Dashboard

```bash
# Every 30 seconds, collect this:
curl http://localhost:8081/api/v1/kubernetes/namespace/summary
curl http://localhost:8081/api/v1/metrics/services
curl http://localhost:8081/api/v1/decisions/stats

# Display:
# - Pod health
# - Resource usage trends
# - Automation actions taken
```

---

## 💡 Key Takeaways

| Component | Purpose | Key Endpoints |
|-----------|---------|---------------|
| **Health System** | Ensure AURA is working | `/health`, `/ready` |
| **Metrics API** | Performance monitoring | `/metrics/:service`, `/history` |
| **Decision API** | Automation audit trail | `/decisions`, `/decisions/:id` |
| **Kubernetes API** | Pod and event tracking | `/kubernetes/pods`, `/events` |
| **Prometheus API** | Metric source integration | `/prometheus/health`, `/query` |
| **Observer** | Monitoring system status | `/observer/health`, `/observer/metrics` |

---

## 🎯 Why AURA Needs All This

```
┌─────────────────────────────────────────────────┐
│ What AURA Does with These Endpoints             │
├─────────────────────────────────────────────────┤
│                                                 │
│ 1. COLLECT DATA                                 │
│    └─ Queries /metrics, /kubernetes, /prometheus
│                                                 │
│ 2. ANALYZE PATTERNS                             │
│    └─ Uses /metrics/history for trends         │
│                                                 │
│ 3. MAKE DECISIONS                               │
│    └─ Decision engine reasons about data        │
│                                                 │
│ 4. TAKE ACTION                                  │
│    └─ Scales pods, restarts services, etc.     │
│                                                 │
│ 5. LEARN & IMPROVE                              │
│    └─ Reviews /decisions for feedback          │
│                                                 │
│ 6. REPORT & AUDIT                               │
│    └─ Provides audit trail via /decisions      │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start Testing

```bash
# Terminal 1: Start AURA
./scripts/setup-kubeconfig.sh

# Terminal 2: Test endpoints
# Check health
curl http://localhost:8081/health

# Get pods
curl http://localhost:8081/api/v1/kubernetes/pods

# Get decisions
curl http://localhost:8081/api/v1/decisions

# Monitor in real-time
while true; do
  curl http://localhost:8081/api/v1/kubernetes/namespace/summary
  sleep 10
done
```

---

## 📚 API Summary Table

| # | Endpoint | Method | Purpose | Status Code |
|---|----------|--------|---------|------------|
| 1 | `/health` | GET | System health | 200/503 |
| 2 | `/ready` | GET | Readiness check | 200/503 |
| 3 | `/metrics` | GET | AURA metrics | 200 |
| 4 | `/api/v1/status` | GET | Service info | 200 |
| 5 | `/api/v1/metrics/:service` | GET | Service metrics | 200/404 |
| 6 | `/api/v1/metrics/:service/:metric/stats` | GET | Metric stats | 200/500 |
| 7 | `/api/v1/metrics/:service/history` | GET | Historical data | 200/404 |
| 8 | `/api/v1/metrics/services` | GET | List services | 200 |
| 9 | `/api/v1/decisions` | GET | Recent decisions | 200 |
| 10 | `/api/v1/decisions/stats` | GET | Decision stats | 200 |
| 11 | `/api/v1/decisions/:id` | GET | Single decision | 200/404 |
| 12 | `/api/v1/observer/health` | GET | Observer status | 200 |
| 13 | `/api/v1/observer/metrics` | GET | Observer metrics | 200 |
| 14 | `/api/v1/kubernetes/pods` | GET | List pods | 200/503 |
| 15 | `/api/v1/kubernetes/pods/:name` | GET | Pod details | 200/404 |
| 16 | `/api/v1/kubernetes/pods/:name/metrics` | GET | Pod history | 200/404 |
| 17 | `/api/v1/kubernetes/events` | GET | All events | 200 |
| 18 | `/api/v1/kubernetes/events/:podname` | GET | Pod events | 200 |
| 19 | `/api/v1/kubernetes/namespace/summary` | GET | Namespace health | 200/503 |
| 20 | `/api/v1/prometheus/health` | GET | Prometheus status | 200/503 |
| 21 | `/api/v1/prometheus/targets` | GET | Scrape targets | 200 |
| 22 | `/api/v1/prometheus/query` | GET | Query metrics | 200/500 |
| 23 | `/api/v1/prometheus/metrics/summary` | GET | Metrics summary | 200 |

That's **23 powerful endpoints** ready to automate your infrastructure! 🚀
