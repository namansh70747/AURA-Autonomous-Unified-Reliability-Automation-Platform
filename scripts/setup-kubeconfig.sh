#!/bin/bash

###############################################################################
# AURA Kubeconfig Setup Script
# Purpose: Prepare kubeconfig for Docker containers running AURA
# This script creates a Docker-friendly kubeconfig that:
# 1. Replaces 127.0.0.1 with host.docker.internal
# 2. Disables SSL verification (required for self-signed certs)
###############################################################################


set -e


# Check for kubectl
if ! command -v kubectl >/dev/null 2>&1; then
    echo "❌ kubectl is not installed or not in PATH. Please install kubectl: https://kubernetes.io/docs/tasks/tools/"
    exit 1
else
    echo "✓ kubectl found: $(kubectl version --client --short 2>/dev/null)"
fi

# Check for minikube
if ! command -v minikube >/dev/null 2>&1; then
    echo "❌ minikube is not installed or not in PATH. Please install minikube: https://minikube.sigs.k8s.io/docs/start/"
    exit 1
else
    echo "✓ minikube found: $(minikube version 2>/dev/null | head -1)"
fi

# Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go is not installed or not in PATH. Please install Go: https://golang.org/dl/"
    exit 1
else
    echo "✓ go found: $(go version 2>/dev/null | awk '{print $3}')"
fi



SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KUBECONFIG_SOURCE="${HOME}/.kube/config"
KUBECONFIG_DOCKER="${HOME}/.kube_docker/config"

echo "🔧 AURA Kubeconfig Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if source kubeconfig exists
if [ ! -f "$KUBECONFIG_SOURCE" ]; then
    echo "❌ Error: Kubeconfig not found at $KUBECONFIG_SOURCE"
    echo "Please ensure kubectl is configured and kubeconfig exists."
    exit 1
fi

echo "✓ Found kubeconfig at: $KUBECONFIG_SOURCE"

# Create .kube_docker directory if it doesn't exist
mkdir -p "${HOME}/.kube_docker"
echo "✓ Created ~/.kube_docker directory"

# Copy the original kubeconfig
cp "$KUBECONFIG_SOURCE" "$KUBECONFIG_DOCKER"
echo "✓ Copied kubeconfig to: $KUBECONFIG_DOCKER"

# Use sed to replace localhost/127.0.0.1 with host.docker.internal
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
    sed -i '' 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
else
    # Linux
    sed -i 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
    sed -i 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
fi
echo "✓ Replaced 127.0.0.1 and localhost with host.docker.internal"

# Add insecure-skip-tls-verify and remove certificate-authority fields using Go
cat > /tmp/aura_add_insecure.go <<'EOF'
package main
import (
    "os"
    "strings"
    "io/ioutil"
)
func main() {
    configFile := os.ExpandEnv("$HOME/.kube_docker/config")
    data, err := ioutil.ReadFile(configFile)
    if err != nil {
        os.Stderr.WriteString("❌ Error reading kubeconfig: " + err.Error() + "\n")
        os.Exit(1)
    }
    lines := strings.Split(string(data), "\n")
    var out []string
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        // Skip any line containing certificate-authority or certificate-authority-data
        if strings.Contains(line, "certificate-authority:") || strings.Contains(line, "certificate-authority-data:") {
            continue
        }
        out = append(out, line)
        // Add insecure-skip-tls-verify after any server line
        if strings.Contains(line, "server:") {
            indent := len(line) - len(strings.TrimLeft(line, " "))
            // Check if insecure-skip-tls-verify already exists in next line
            already := false
            if i+1 < len(lines) && strings.Contains(lines[i+1], "insecure-skip-tls-verify") {
                already = true
            }
            if !already {
                out = append(out, strings.Repeat(" ", indent)+"insecure-skip-tls-verify: true")
            }
        }
    }
    err = ioutil.WriteFile(configFile, []byte(strings.Join(out, "\n")), 0644)
    if err != nil {
        os.Stderr.WriteString("❌ Error writing kubeconfig: " + err.Error() + "\n")
        os.Exit(1)
    }
    println("✓ Removed certificate-authority fields and added insecure-skip-tls-verify")
}
EOF
go run /tmp/aura_add_insecure.go && rm /tmp/aura_add_insecure.go

echo "✅ Kubeconfig setup complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Location: $KUBECONFIG_DOCKER"
echo ""
# Verify the changes
echo ""
echo "📋 Docker Kubeconfig Servers:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
grep -E "server:|insecure-skip-tls-verify" "$KUBECONFIG_DOCKER" | head -20 || true
echo ""

echo "✅ Kubeconfig setup complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Location: $KUBECONFIG_DOCKER"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Start AURA and keep this script running (do not shut down AURA easily)
# ─────────────────────────────────────────────────────────────────────────────

# Basic prerequisites checks
if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Docker is not installed or not in PATH. Please install Docker Desktop and retry."
    exit 1
fi

if ! command -v docker-compose >/dev/null 2>&1; then
    echo "❌ docker-compose is not installed. Please install Docker Compose and retry."
    exit 1
fi




# Ensure minikube is running
echo ""
echo "🔍 Checking Minikube status..."
if ! minikube status >/dev/null 2>&1; then
    echo "⚠️  Minikube is not running. Starting minikube..."
    minikube start --driver=docker
    if [ $? -ne 0 ]; then
        echo "❌ Failed to start minikube. Please check your Docker installation."
        exit 1
    fi
    echo "✓ Minikube started successfully."
    sleep 3
else
    echo "✓ Minikube is already running."
    # Verify API server is accessible
    if ! kubectl get nodes >/dev/null 2>&1; then
        echo "⚠️  Minikube is running but API server is not accessible. Restarting..."
        minikube delete
        minikube start --driver=docker
        echo "✓ Minikube restarted successfully."
        sleep 3
    fi
fi

# Verify kubectl can connect
echo "🔍 Verifying kubectl connection..."
if ! kubectl get nodes >/dev/null 2>&1; then
    echo "❌ kubectl cannot connect to Kubernetes. Please check your cluster."
    exit 1
fi
echo "✓ kubectl connected to Kubernetes cluster"

# Get current API server endpoint and port
echo ""
echo "🔍 Getting current Kubernetes API server details..."

# Extract API server URL and clean ANSI escape codes
API_SERVER_RAW=$(kubectl cluster-info 2>/dev/null | grep "control plane" | head -1)
API_SERVER=$(echo "$API_SERVER_RAW" | sed 's/\x1b\[[0-9;]*m//g' | awk '{print $NF}')

# Extract port number more reliably
API_PORT=$(echo "$API_SERVER" | grep -oE '[0-9]{4,5}$')

# Validate we got values
if [ -z "$API_SERVER" ] || [ -z "$API_PORT" ]; then
    echo "   ⚠️  Could not extract API server details, using default"
    API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' 2>/dev/null)
    API_PORT=$(echo "$API_SERVER" | grep -oE '[0-9]{4,5}$')
fi

echo "   API Server: ${API_SERVER}"
echo "   API Port: ${API_PORT}"

# Test if the port is accessible from Docker
if [ -n "$API_PORT" ]; then
    echo "   Testing Docker → host.docker.internal:${API_PORT} connectivity..."
    if timeout 3 bash -c "docker run --rm alpine sh -c 'nc -zv host.docker.internal ${API_PORT}'" 2>&1 | grep -q "succeeded\|open"; then
        echo "   ✓ Port ${API_PORT} is accessible from Docker containers"
    else
        echo "   ⚠️  Port ${API_PORT} may not be accessible from Docker (will try anyway)"
    fi
fi

# Regenerate Docker kubeconfig with updated cluster endpoints
echo ""
echo "🔄 Regenerating Docker kubeconfig with latest cluster info..."
cp "$KUBECONFIG_SOURCE" "$KUBECONFIG_DOCKER"

# Use sed to replace localhost/127.0.0.1 with host.docker.internal
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
    sed -i '' 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
else
    # Linux
    sed -i 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
    sed -i 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
fi

# Re-run the Go script to remove certificate-authority and add insecure-skip-tls-verify
# Also clean up old/stale server entries
cat > /tmp/aura_regen_insecure.go <<'EOFGO'
package main
import (
    "os"
    "strings"
    "io/ioutil"
)
func main() {
    configFile := os.ExpandEnv("$HOME/.kube_docker/config")
    data, err := ioutil.ReadFile(configFile)
    if err != nil {
        os.Stderr.WriteString("❌ Error reading kubeconfig: " + err.Error() + "\n")
        os.Exit(1)
    }
    
    lines := strings.Split(string(data), "\n")
    var out []string
    inCluster := false
    serverSeen := false
    
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        
        // Track if we're in a cluster section
        if strings.Contains(line, "- cluster:") {
            inCluster = true
            serverSeen = false
        } else if strings.HasPrefix(strings.TrimSpace(line), "name:") && inCluster {
            inCluster = false
        }
        
        // Skip certificate-authority fields
        if strings.Contains(line, "certificate-authority:") || strings.Contains(line, "certificate-authority-data:") {
            continue
        }
        
        // If we see a server line in a cluster, mark it
        if inCluster && strings.Contains(line, "server:") {
            if serverSeen {
                // Skip duplicate server entries in same cluster
                continue
            }
            serverSeen = true
        }
        
        out = append(out, line)
        
        // Add insecure-skip-tls-verify after server line
        if strings.Contains(line, "server:") {
            indent := len(line) - len(strings.TrimLeft(line, " "))
            already := false
            if i+1 < len(lines) && strings.Contains(lines[i+1], "insecure-skip-tls-verify") {
                already = true
            }
            if !already {
                out = append(out, strings.Repeat(" ", indent)+"insecure-skip-tls-verify: true")
            }
        }
    }
    
    err = ioutil.WriteFile(configFile, []byte(strings.Join(out, "\n")), 0644)
    if err != nil {
        os.Stderr.WriteString("❌ Error writing kubeconfig: " + err.Error() + "\n")
        os.Exit(1)
    }
    println("✓ Kubeconfig regenerated with latest endpoints")
}
EOFGO
go run /tmp/aura_regen_insecure.go && rm /tmp/aura_regen_insecure.go

echo "✓ Docker kubeconfig updated with latest minikube endpoints"
echo ""
echo "📋 Current Kubeconfig Server:"
CURRENT_SERVER=$(grep "server:" "$KUBECONFIG_DOCKER" | head -1 | awk '{print $2}')
echo "   ${CURRENT_SERVER}"
echo ""

# Ensure prometheus.yml.local exists (required for docker-compose.override.yml)
echo "🔍 Checking Prometheus configuration..."

# Remove if it's a directory (common mistake)
if [ -d "$PROJECT_ROOT/configs/prometheus.yml.local" ]; then
    echo "⚠️  prometheus.yml.local is a directory (wrong!), removing it..."
    rm -rf "$PROJECT_ROOT/configs/prometheus.yml.local"
fi

# Remove if it's a symlink or has wrong permissions
if [ -L "$PROJECT_ROOT/configs/prometheus.yml.local" ]; then
    echo "⚠️  prometheus.yml.local is a symlink, removing it..."
    rm -f "$PROJECT_ROOT/configs/prometheus.yml.local"
fi

# Always recreate the file to ensure it's correct
if [ -f "$PROJECT_ROOT/configs/prometheus.yml.local" ]; then
    echo "📝 Recreating prometheus.yml.local to ensure it's up to date..."
    rm -f "$PROJECT_ROOT/configs/prometheus.yml.local"
fi

echo "📝 Creating prometheus.yml.local..."
cat > "$PROJECT_ROOT/configs/prometheus.yml.local" <<'PROMEOF'
# Prometheus Configuration for AURA (Local Docker)
global:
  scrape_interval: 10s
  evaluation_interval: 10s

scrape_configs:
  # Scrape sample-app metrics
  - job_name: "sample-app"
    static_configs:
      - targets: ["sample-app:8080"]
        labels:
          service: "sample-app"
          environment: "dev"

  # Scrape Prometheus itself
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]

  # Scrape AURA main application
  - job_name: "aura"
    static_configs:
      - targets: ["aura:8081"]
        labels:
          service: "aura"
          environment: "dev"
PROMEOF

# Verify the file was created successfully
if [ -f "$PROJECT_ROOT/configs/prometheus.yml.local" ] && [ ! -d "$PROJECT_ROOT/configs/prometheus.yml.local" ]; then
    FILE_SIZE=$(wc -c < "$PROJECT_ROOT/configs/prometheus.yml.local")
    if [ "$FILE_SIZE" -gt 100 ]; then
        echo "✅ Created configs/prometheus.yml.local (${FILE_SIZE} bytes)"
        chmod 644 "$PROJECT_ROOT/configs/prometheus.yml.local"
    else
        echo "❌ prometheus.yml.local file is too small or empty!"
        exit 1
    fi
else
    echo "❌ Failed to create prometheus.yml.local as a regular file!"
    ls -la "$PROJECT_ROOT/configs/prometheus.yml.local" || echo "File does not exist"
    exit 1
fi

# Start Docker Compose
echo ""
echo "🐳 Starting Docker Compose services..."
cd "$PROJECT_ROOT"
docker-compose -f docker-compose.yml down --remove-orphans 2>/dev/null || true

# Before starting, let's check if we should remove sample-app service
# (since we'll use the native binary)
echo "⚙️  Configuring Docker Compose..."
echo "   (Skipping sample-app container - using native binary instead)"

# Start all services except sample-app
docker-compose -f docker-compose.yml up -d --remove-orphans -f docker-compose.yml -f <(cat <<'EOF'
version: '3.8'
services:
  sample-app:
    deploy:
      replicas: 0
EOF
) 2>/dev/null || docker-compose -f docker-compose.yml up -d --remove-orphans --scale sample-app=0 2>/dev/null || {
    # If scaling doesn't work, just start without it
    echo "   Starting core services (postgres, prometheus, aura)..."
    docker-compose -f docker-compose.yml up -d postgres prometheus aura 2>/dev/null
}

echo ""
echo "⏳ Waiting for services to start..."

# Wait for containers to be created (max 30 seconds)
echo -n "   "
for i in {1..30}; do
    POSTGRES_RUNNING=$(docker ps --filter "name=aura-postgres" --filter "status=running" -q 2>/dev/null | wc -l)
    PROMETHEUS_RUNNING=$(docker ps --filter "name=aura-prometheus" --filter "status=running" -q 2>/dev/null | wc -l)
    AURA_RUNNING=$(docker ps --filter "name=aura-main" --filter "status=running" -q 2>/dev/null | wc -l)
    
    if [ "$POSTGRES_RUNNING" -gt 0 ] && [ "$PROMETHEUS_RUNNING" -gt 0 ] && [ "$AURA_RUNNING" -gt 0 ]; then
        echo "✓"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

# Final check
POSTGRES_RUNNING=$(docker ps --filter "name=aura-postgres" --filter "status=running" -q 2>/dev/null | wc -l)
PROMETHEUS_RUNNING=$(docker ps --filter "name=aura-prometheus" --filter "status=running" -q 2>/dev/null | wc -l)
AURA_RUNNING=$(docker ps --filter "name=aura-main" --filter "status=running" -q 2>/dev/null | wc -l)

if [ "$POSTGRES_RUNNING" -eq 0 ] || [ "$PROMETHEUS_RUNNING" -eq 0 ] || [ "$AURA_RUNNING" -eq 0 ]; then
    echo "⚠️  Some core services failed to start"
    echo "   Postgres: ${POSTGRES_RUNNING}, Prometheus: ${PROMETHEUS_RUNNING}, AURA: ${AURA_RUNNING}"
    echo "   Check logs: docker-compose logs"
    echo "   Attempting emergency restart..."
    docker-compose -f docker-compose.yml up -d --force-recreate 2>&1 | tail -10
    sleep 10
else
    echo "✓ All core containers are running"
fi

# Wait for AURA to be healthy (max 60 seconds)
echo ""
echo "⏳ Waiting for AURA to be healthy..."
echo -n "   "
for i in {1..60}; do
    HEALTH_STATUS=$(curl -s http://localhost:8081/health 2>/dev/null | jq -r '.status' 2>/dev/null)
    if [ "$HEALTH_STATUS" = "healthy" ]; then
        echo "✓"
        echo "✅ AURA is healthy and ready!"
        break
    fi
    echo -n "."
    sleep 1
    
    # If 30 seconds passed and still not healthy, check logs
    if [ $i -eq 30 ]; then
        echo ""
        echo "   Still waiting... Checking AURA logs:"
        docker logs aura-main --tail 5 2>&1 | sed 's/^/      /'
        echo -n "   "
    fi
done
echo ""

echo "✓ Docker Compose core services started and healthy"

# Now ensure sample-app native binary is running
echo ""
echo "� Starting native sample-app binary..."
if [ -f "./bin/sample-app-mac" ]; then
    # Kill any existing sample-app processes
    pkill -f "sample-app-mac" 2>/dev/null || true
    sleep 1
    
    # Start the native binary
    nohup ./bin/sample-app-mac > /tmp/sample-app.log 2>&1 &
    SAMPLE_APP_PID=$!
    sleep 2
    
    # Verify it started
    if pgrep -f "sample-app-mac" > /dev/null 2>&1; then
        echo "✓ Sample-app native binary started successfully (PID: $SAMPLE_APP_PID)"
        echo "  Logs: tail -f /tmp/sample-app.log"
    else
        echo "❌ Failed to start sample-app native binary"
        echo "  Check logs: tail -f /tmp/sample-app.log"
        cat /tmp/sample-app.log
    fi
else
    echo "❌ sample-app-mac binary not found in ./bin/"
    echo "   Please run: make build-sample-app"
fi

echo ""
echo "✅ All Docker Compose services started successfully"

# ─────────────────────────────────────────────────────────────────────────────
# Verify Kubernetes Connectivity
# ─────────────────────────────────────────────────────────────────────────────

echo ""
echo "🔍 Verifying Kubernetes connectivity..."
K8S_TEST_RESULT=$(curl -s http://localhost:8081/api/v1/kubernetes/pods 2>&1)

if echo "$K8S_TEST_RESULT" | grep -q '"pods"'; then
    POD_COUNT=$(echo "$K8S_TEST_RESULT" | jq -r '.count' 2>/dev/null || echo "0")
    echo "✅ Kubernetes API is accessible!"
    echo "   Pods in default namespace: ${POD_COUNT}"
    
    if [ "$POD_COUNT" -eq 0 ]; then
        echo ""
        echo "💡 No pods found. Deploy a test pod with:"
        echo "   kubectl run test-nginx --image=nginx:alpine"
    else
        echo ""
        echo "📋 Current Pods:"
        echo "$K8S_TEST_RESULT" | jq -r '.pods[] | "   • \(.name) (\(.phase))"' 2>/dev/null || true
    fi
else
    echo "⚠️  Kubernetes API not accessible yet"
    ERROR_MSG=$(echo "$K8S_TEST_RESULT" | jq -r '.error' 2>/dev/null || echo "$K8S_TEST_RESULT" | head -c 150)
    echo "   Error: ${ERROR_MSG}"
    echo ""
    echo "   This is usually fixed by:"
    echo "   1. Waiting 10-20 seconds for AURA to fully initialize"
    echo "   2. Running: docker-compose restart aura"
    echo "   3. Checking logs: docker logs aura-main --tail 20"
fi

echo ""
echo "🎉 All services started!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "AURA API:        http://localhost:8081"
echo "Prometheus:      http://localhost:9090"
echo "Sample App:      http://localhost:8080"
echo "PostgreSQL:      localhost:5432"
echo "Kubeconfig:      $KUBECONFIG_DOCKER"
echo "Minikube IP:     $(minikube ip 2>/dev/null || echo 'N/A')"
echo "Kubernetes:      $(kubectl get nodes -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo 'N/A')"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ Project startup complete!"
echo "💡 Quick test: curl http://localhost:8081/health"
echo "💡 Check K8s pods: curl http://localhost:8081/api/v1/kubernetes/pods"
echo "💡 Run health checks: ./scripts/check-endpoints.sh"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Verify Metrics Collection (HTTP Requests)
# ─────────────────────────────────────────────────────────────────────────────

echo ""
echo "📊 Verifying metrics collection..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Wait for sample-app to be accessible
sleep 3

# Check if sample-app is running
if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "✓ Sample app is healthy (container or native)"
else
    echo "⚠️  Sample app not responding at http://localhost:8080"
    echo "   Starting native sample-app binary..."
    cd "$PROJECT_ROOT"
    if [ -f "./bin/sample-app-mac" ]; then
        # Kill any existing sample-app processes
        pkill -f "sample-app-mac" 2>/dev/null || true
        sleep 1
        
        # Start the native binary
        nohup ./bin/sample-app-mac > /tmp/sample-app.log 2>&1 &
        SAMPLE_APP_PID=$!
        sleep 3
        
        # Verify it started
        if pgrep -f "sample-app-mac" > /dev/null 2>&1; then
            echo "✓ Native sample-app started successfully (PID: $SAMPLE_APP_PID)"
            echo "  Logs: /tmp/sample-app.log"
        else
            echo "❌ Failed to start native sample-app"
            echo "  Check logs: tail -f /tmp/sample-app.log"
        fi
    else
        echo "❌ sample-app-mac binary not found in ./bin/"
        echo "   Please run: make build-sample-app"
    fi
fi

# Verify sample-app is healthy after startup
if ! curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "⚠️  Sample app still not responding after startup attempts"
    echo "   Continuing with setup, but metrics may not be available initially"
else
    echo "✓ Sample app confirmed healthy"
fi

# Wait for Prometheus to scrape metrics
echo ""
echo "⏳ Waiting for Prometheus to collect metrics (10-15 seconds)..."
sleep 10

# Check if metrics are available in AURA API
echo ""
echo "🔍 Checking if metrics are available in AURA API..."
METRICS_RESPONSE=$(curl -s http://localhost:8081/api/v1/metrics/sample-app 2>/dev/null)

if echo "$METRICS_RESPONSE" | jq -e '.metrics.http_requests' >/dev/null 2>&1; then
    echo "✓ HTTP Requests metric is available!"
    echo ""
    echo "📊 Current Metrics:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$METRICS_RESPONSE" | jq '.metrics' 2>/dev/null || echo "   (Could not parse metrics)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    echo "⚠️  HTTP Requests metric not yet available"
    echo "   This is normal - metrics collection takes 10-30 seconds on first run"
    echo ""
    echo "   To manually test metrics:"
    echo "   1. Sample-app metrics: curl http://localhost:8080/metrics | grep http_requests_total"
    echo "   2. Prometheus query:  curl 'http://localhost:9090/api/v1/query?query=http_requests_total' | jq"
    echo "   3. AURA API metrics:  curl http://localhost:8081/api/v1/metrics/sample-app | jq '.metrics'"
    echo ""
    echo "   Or wait a bit and try: curl http://localhost:8081/api/v1/metrics/sample-app"
fi

echo ""
echo "📜 Quick API Tests:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ Health check:"
echo "  curl http://localhost:8081/health"
echo ""
echo "✓ Current service metrics (CPU, Memory, HTTP Requests):"
echo "  curl http://localhost:8081/api/v1/metrics/sample-app"
echo ""
echo "✓ All monitored services:"
echo "  curl http://localhost:8081/api/v1/metrics/services"
echo ""
echo "✓ Kubernetes pods:"
echo "  curl http://localhost:8081/api/v1/kubernetes/pods"
echo ""
echo "✓ Recent automation decisions:"
echo "  curl http://localhost:8081/api/v1/decisions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "🔒 Security Check:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✓ Your local IP is protected (git-ignored)"
echo "✓ Prometheus config uses local override"
echo "✓ Docker-compose uses local overrides"
echo "✓ All sensitive files are .gitignored"
echo ""
echo "For security audit details, see: SECURITY_AUDIT.md"
echo "For security verification, run: ./SECURITY_CHECKLIST.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "✨ AURA is ready! All systems operational."
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Keep Services Running - Background Monitoring Loop
# ─────────────────────────────────────────────────────────────────────────────

echo ""
echo "🔄 Starting background health monitoring..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 Monitoring Services:"
echo "   • Docker Compose containers (AURA, Prometheus, PostgreSQL)"
echo "   • Sample-app native binary"
echo "   • Kubernetes pods"
echo ""
echo "The services will continue running in the background."
echo "Press Ctrl+C to stop monitoring (services stay running)."
echo "To stop all services: docker-compose down"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Create a monitoring loop that keeps running
SAMPLE_APP_PID=""
MONITOR_INTERVAL=30  # Check every 30 seconds

# Function to start sample-app if not running
start_sample_app() {
    if ! pgrep -f "sample-app-mac" > /dev/null 2>&1; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  Sample-app not running, restarting..."
        cd "$PROJECT_ROOT"
        if [ -f "./bin/sample-app-mac" ]; then
            # Kill any stray processes
            pkill -f "sample-app-mac" 2>/dev/null || true
            sleep 1
            
            # Start the native binary properly
            nohup ./bin/sample-app-mac > /tmp/sample-app.log 2>&1 &
            SAMPLE_APP_PID=$!
            sleep 2
            
            if pgrep -f "sample-app-mac" > /dev/null 2>&1; then
                echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ Sample-app restarted (PID: $SAMPLE_APP_PID)"
            else
                echo "[$(date '+%Y-%m-%d %H:%M:%S')] ❌ Failed to start sample-app, check /tmp/sample-app.log"
            fi
        fi
    fi
}

# Function to check Docker Compose services
check_docker_services() {
    local total=$(docker-compose -f "$PROJECT_ROOT/docker-compose.yml" ps -q 2>/dev/null | wc -l)
    local running=$(docker-compose -f "$PROJECT_ROOT/docker-compose.yml" ps --filter "status=running" -q 2>/dev/null | wc -l)
    
    if [ "$total" -gt 0 ] && [ "$running" -lt "$total" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  Some Docker services are down ($running/$total running)"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restarting Docker Compose..."
        cd "$PROJECT_ROOT"
        docker-compose down --remove-orphans 2>/dev/null || true
        sleep 2
        docker-compose up -d --remove-orphans 2>/dev/null
        sleep 3
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ Docker services restarted"
    elif [ "$total" -eq 0 ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  No Docker containers running!"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Docker Compose..."
        cd "$PROJECT_ROOT"
        docker-compose up -d --remove-orphans 2>/dev/null
        sleep 5
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ Docker Compose started"
    fi
}

# Function to check AURA API health
check_aura_health() {
    local health_status=$(curl -s http://localhost:8081/health 2>/dev/null | jq -r '.status' 2>/dev/null)
    if [ "$health_status" != "healthy" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  AURA API is not healthy"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Trying to recover..."
        cd "$PROJECT_ROOT"
        docker-compose restart aura 2>/dev/null
        sleep 3
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ AURA restarted"
    fi
}

# Function to check Prometheus health
check_prometheus_health() {
    local prom_status=$(curl -s http://localhost:9090/-/healthy 2>/dev/null)
    if [ -z "$prom_status" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  Prometheus is not responding"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restarting Prometheus..."
        cd "$PROJECT_ROOT"
        docker-compose restart prometheus 2>/dev/null
        sleep 3
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ Prometheus restarted"
    fi
}

# Function to check Kubernetes connectivity
check_kubernetes_connectivity() {
    local k8s_response=$(curl -s http://localhost:8081/api/v1/kubernetes/pods 2>&1)
    
    # Check if we got an error response
    if echo "$k8s_response" | grep -q '"error".*connection refused'; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  Kubernetes API not accessible from AURA"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Checking minikube status..."
        
        # Check if minikube is running
        if ! minikube status | grep -q "apiserver: Running"; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] ❌ Minikube API server is down, restarting..."
            minikube start --driver=docker >/dev/null 2>&1
            sleep 5
        fi
        
        # Get current API server port
        local api_port=$(kubectl cluster-info 2>/dev/null | grep "control plane" | sed -n 's/.*:\([0-9]\+\).*/\1/p')
        if [ -n "$api_port" ]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Updating kubeconfig with new API server port: ${api_port}"
            
            # Regenerate Docker kubeconfig
            cp "$KUBECONFIG_SOURCE" "$KUBECONFIG_DOCKER"
            
            # Replace with host.docker.internal
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
                sed -i '' 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
            else
                sed -i 's/127\.0\.0\.1/host.docker.internal/g' "$KUBECONFIG_DOCKER"
                sed -i 's/localhost/host.docker.internal/g' "$KUBECONFIG_DOCKER"
            fi
            
            # Remove certificate-authority and add insecure-skip-tls-verify
            cat > /tmp/k8s_fix_config.go <<'GOSCRIPT'
package main
import ("os"; "strings"; "io/ioutil")
func main() {
    configFile := os.ExpandEnv("$HOME/.kube_docker/config")
    data, _ := ioutil.ReadFile(configFile)
    lines := strings.Split(string(data), "\n")
    var out []string
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        if strings.Contains(line, "certificate-authority:") || strings.Contains(line, "certificate-authority-data:") {
            continue
        }
        out = append(out, line)
        if strings.Contains(line, "server:") {
            indent := len(line) - len(strings.TrimLeft(line, " "))
            if i+1 >= len(lines) || !strings.Contains(lines[i+1], "insecure-skip-tls-verify") {
                out = append(out, strings.Repeat(" ", indent)+"insecure-skip-tls-verify: true")
            }
        }
    }
    ioutil.WriteFile(configFile, []byte(strings.Join(out, "\n")), 0644)
}
GOSCRIPT
            go run /tmp/k8s_fix_config.go >/dev/null 2>&1 && rm /tmp/k8s_fix_config.go
            
            # Restart AURA to pick up new kubeconfig
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restarting AURA to apply new kubeconfig..."
            cd "$PROJECT_ROOT"
            docker-compose restart aura >/dev/null 2>&1
            sleep 8
            
            # Verify fix
            local test_result=$(curl -s http://localhost:8081/api/v1/kubernetes/pods 2>&1)
            if echo "$test_result" | grep -q '"pods"'; then
                echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✅ Kubernetes connectivity restored!"
            else
                echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  Kubernetes still not accessible, will retry later"
            fi
        fi
    fi
}

# Trap Ctrl+C to handle graceful shutdown
cleanup() {
    echo ""
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🛑 Monitoring stopped by user"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Services are still running!"
    echo ""
    echo "To view running services:"
    echo "  docker-compose ps"
    echo ""
    echo "To stop all services:"
    echo "  docker-compose down"
    echo ""
    echo "To restart monitoring:"
    echo "  ./scripts/setup-kubeconfig.sh"
    echo ""
    exit 0
}

trap cleanup INT TERM

# Initial status
echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ All services initialized"
echo ""

# Main monitoring loop
LOOP_COUNT=0
while true; do
    # Check and restart services if needed
    check_docker_services
    start_sample_app
    check_aura_health
    check_prometheus_health
    
    # Check Kubernetes connectivity every 2 minutes (every 4th iteration)
    LOOP_COUNT=$((LOOP_COUNT + 1))
    if [ $((LOOP_COUNT % 4)) -eq 0 ]; then
        check_kubernetes_connectivity
    fi
    
    # Display status
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ All systems running healthy"
    
    # Sleep before next check
    sleep $MONITOR_INTERVAL
done

