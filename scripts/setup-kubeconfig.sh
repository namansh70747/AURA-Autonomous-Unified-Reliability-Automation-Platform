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
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        if strings.Contains(line, "certificate-authority:") || strings.Contains(line, "certificate-authority-data:") {
            continue
        }
        out = append(out, line)
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
echo "📋 Updated Kubeconfig Servers:"
grep -E "server:" "$KUBECONFIG_DOCKER" | head -5 || true
echo ""

# Start Docker Compose
echo ""
echo "🐳 Starting Docker Compose services..."
cd "$PROJECT_ROOT"
docker-compose -f docker-compose.yml down --remove-orphans 2>/dev/null || true
docker-compose -f docker-compose.yml up -d --remove-orphans

echo ""
echo "⏳ Waiting for services to be healthy..."
sleep 5

# Check if containers are running
CONTAINERS=$(docker-compose ps -q 2>/dev/null | wc -l)
if [ "$CONTAINERS" -eq 0 ]; then
    echo "❌ No containers are running. Please check docker-compose.yml"
    exit 1
fi

echo "✓ Docker Compose services started successfully"

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
echo "📜 Tailing AURA logs (press Ctrl+C to stop, services will keep running)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
docker-compose -f docker-compose.yml logs -f aura
