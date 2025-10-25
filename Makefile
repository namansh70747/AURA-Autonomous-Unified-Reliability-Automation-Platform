.PHONY: help setup build-linux test clean docker-up docker-down logs init db-reset verify setup-kubeconfig build-all k8-start k8-metrics

# Default target
help:
	@echo "🧠 AURA - Autonomous Unified Reliability Automation"
	@echo "================================================"
	@echo "QUICK START (Recommended - Docker + Kubernetes):"
	@echo "  make setup-kubeconfig  - Setup kubeconfig for Docker"
	@echo "  make docker-up         - Start AURA with Docker"
	@echo "  make test-endpoints    - Test all API endpoints"
	@echo "  make k8-start          - Start AURA with Kubernetes (alias)"
	@echo "  make k8-metrics        - View Kubernetes metrics (alias)"
	@echo ""
	@echo "BUILDING:"
	@echo "  setup         - Install dependencies"
	@echo "  build-linux   - Build Linux binaries for Docker"
	@echo "  build-all     - Build AURA + sample-app"
	@echo ""
	@echo "DOCKER:"
	@echo "  docker-up     - Start all Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  docker-rebuild- Rebuild and restart services"
	@echo "  logs          - Show all Docker logs"
	@echo ""
	@echo "KUBERNETES:"
	@echo "  k8s-deploy    - Deploy to Kubernetes cluster"
	@echo "  k8s-start     - Start Kubernetes with metrics scraping"
	@echo "  k8s-metrics   - View real-time Kubernetes metrics"
	@echo "  k8s-status    - Show Kubernetes status"
	@echo "  k8s-clean     - Clean Kubernetes resources"
	@echo "  k8-start      - Alias for k8s-start (start with Kubernetes)"
	@echo "  k8-metrics    - Alias for k8s-metrics (view metrics)"
	@echo ""
	@echo "TESTING:"
	@echo "  test          - Run Go tests"
	@echo "  test-all      - Test all endpoints"
	@echo "  health        - Check system health"
	@echo ""
	@echo "UTILITIES:"
	@echo "  clean         - Clean build artifacts"
	@echo "  db-reset      - Reset database"
	@echo "  verify        - Verify prerequisites"
	@echo "================================================"

# Install dependencies
setup:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Build AURA
build-linux:
	@echo "🔨 Building Linux binaries for Docker..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/aura-linux cmd/aura/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sample-app-linux examples/sample-app/main.go
	@echo "✅ Linux binaries built"
	@echo "   📦 bin/aura-linux"
	@echo "   � bin/sample-app-linux"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Start Docker services (PostgreSQL + Prometheus + Sample App + AURA)
docker-up:
	@echo "🐳 Starting AURA with Docker..."
	docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ All services started:"
	@echo "   � AURA API:       http://localhost:8081"
	@echo "   �📊 Prometheus:     http://localhost:9090"
	@echo "   💾 PostgreSQL:     localhost:5432"
	@echo "   🌐 Sample App:     http://localhost:8080"
	@echo ""
	@echo "Test endpoints with: make test-endpoints"

# Setup kubeconfig for Docker (must run once)
setup-kubeconfig:
	@echo "Setting up kubeconfig for Docker containers..."
	@./scripts/setup-kubeconfig.sh

# Stop Docker services
docker-down:
	@echo "🛑 Stopping Docker services..."
	docker-compose down
	@echo "✅ Services stopped"

# Show Docker logs
logs:
	docker-compose logs -f

# Show specific service logs
logs-postgres:
	docker-compose logs -f postgres

logs-prometheus:
	docker-compose logs -f prometheus

logs-sample:
	docker-compose logs -f sample-app

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf bin/aura bin/sample-app
	@echo "✅ Build artifacts cleaned"

# Clean everything including Docker volumes
clean-all:
	@echo "🧹 Deep cleaning..."
	@rm -rf bin/
	@docker-compose down -v
	@echo "✅ Everything cleaned"

# Reset database (drop and recreate)
db-reset:
	@echo "🔄 Resetting database..."
	docker-compose down -v
	docker-compose up -d postgres
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@until docker exec aura-postgres pg_isready -U aura -d aura_db; do sleep 2; done
	@sleep 5
	@echo "✅ Database reset complete"

# Full initialization (first time setup)
init: setup docker-up
	@sleep 10
	@echo "🎉 AURA is ready!"
	@echo "Run 'make run' to start AURA"

# Verify setup
verify:
	@echo "🔍 Verifying setup..."
	@echo "Checking Go version..."
	@go version
	@echo "Checking Docker..."
	@docker --version
	@echo "Checking Docker Compose..."
	@docker-compose version
	@echo "✅ All prerequisites verified"

# Quick start - Complete setup and run
start: setup-kubeconfig build-linux docker-up
	@echo "🎉 AURA platform is running!"
	@echo "✅ All services initialized and ready"

# Test all endpoints
test-endpoints:
	@echo "🧪 Testing AURA API Endpoints..."
	@echo ""
	@echo "Status Endpoints:"
	@curl -s http://localhost:8081/health | jq . && echo ""
	@curl -s http://localhost:8081/ready | jq . && echo ""
	@curl -s http://localhost:8081/api/v1/status | jq . && echo ""
	@echo "Kubernetes Endpoints:"
	@curl -s http://localhost:8081/api/v1/kubernetes/pods | jq . && echo ""
	@echo "Prometheus Endpoints:"
	@curl -s http://localhost:8081/api/v1/prometheus/health | jq . && echo ""
	@echo "✅ All endpoints tested"

# Development workflow
dev: docker-up build run

# Production build
prod-build:
	@echo "🏭 Building production binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/aura-linux cmd/aura/main.go
	@echo "✅ Production build complete: bin/aura-linux"

# Kubernetes deployment
k8s-deploy:
	@echo "☸️  Deploying to Kubernetes..."
	@echo "Starting minikube (if not running)..."
	@minikube status || minikube start
	@echo "Loading Docker images into minikube..."
	@eval $$(minikube docker-env) && docker-compose build
	@echo "Deploying PostgreSQL..."
	@kubectl apply -f deployments/kubernetes/postgres.yaml
	@echo "Deploying Prometheus..."
	@kubectl apply -f deployments/kubernetes/prometheus.yaml
	@echo "Deploying RBAC..."
	@kubectl apply -f deployments/kubernetes/rbac.yaml
	@echo "Deploying test app..."
	@kubectl apply -f deployments/kubernetes/test-app.yaml
	@echo "⏳ Waiting for pods to be ready..."
	@sleep 10
	@kubectl get pods -A
	@echo "✅ Kubernetes deployment complete"

# Start Kubernetes with metrics scraping enabled
k8s-start: k8s-deploy build
	@echo "🎯 Starting AURA with Kubernetes metrics scraping..."
	@echo "================================================"
	@echo "Kubernetes cluster is ready"
	@echo "AURA will scrape metrics from:"
	@echo "  • Kubernetes API (pod metrics)"
	@echo "  • Prometheus (application metrics)"
	@echo "================================================"
	@echo ""
	@echo "Starting AURA with Kubernetes watcher enabled..."
	@./bin/aura

# View Kubernetes metrics in real-time
k8s-metrics:
	@echo "📊 Real-time Kubernetes Metrics"
	@echo "================================================"
	@echo ""
	@echo "📦 Pod Status:"
	@kubectl get pods -o wide
	@echo ""
	@echo "📈 Pod Resource Usage:"
	@kubectl top pods 2>/dev/null || echo "⚠️  Metrics server not installed. Run: minikube addons enable metrics-server"
	@echo ""
	@echo "🔍 Recent Events:"
	@kubectl get events --sort-by=.metadata.creationTimestamp | tail -10
	@echo ""
	@echo "💾 Database Metrics Count:"
	@docker exec aura-postgres psql -U aura -d aura_db -c "SELECT COUNT(*) as total_metrics FROM metrics;" 2>/dev/null || echo "⚠️  Database not accessible"
	@echo ""
	@echo "================================================"

# Alias for k8s-metrics
k8-metrics: k8s-metrics

# Alias for k8s-start
k8-start: k8s-start

# Show Kubernetes status
k8s-status:
	@echo "☸️  Kubernetes Status"
	@echo "================================================"
	@echo "Cluster Info:"
	@kubectl cluster-info
	@echo ""
	@echo "Pods:"
	@kubectl get pods
	@echo ""
	@echo "Services:"
	@kubectl get svc
	@echo ""
	@echo "Deployments:"
	@kubectl get deployments
	@echo "================================================"

# Clean Kubernetes resources
k8s-clean:
	@echo "🧹 Cleaning Kubernetes resources..."
	@kubectl delete -f deployments/kubernetes/test-app.yaml --ignore-not-found=true
	@kubectl delete -f deployments/kubernetes/postgres.yaml --ignore-not-found=true
	@echo "✅ Kubernetes resources cleaned"

# Day 7 Integration Test
integration-test: k8s-deploy build
	@echo "🧪 Running Day 7 Integration Test..."
	@echo "================================================"
	@echo "1. Kubernetes cluster running"
	@echo "2. Test app deployed"
	@echo "3. Starting AURA to observe metrics..."
	@echo "================================================"
	@echo "Run: ./bin/aura"
	@echo "Expected: AURA observes pods every 10 seconds"
	@echo "         Metrics stored in PostgreSQL"
	@echo "         Console output shows pod status"
	@echo "================================================"

# Rebuild and restart Docker services
docker-rebuild:
	@echo "🔄 Rebuilding and restarting Docker services..."
	docker-compose down
	docker-compose up -d --build
	@echo "⏳ Waiting for services..."
	@sleep 15
	@echo "✅ Services restarted"
	@make health

# Check system health
health:
	@echo "🏥 Checking system health..."
	@echo ""
	@echo "1. Container Status:"
	@docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "aura|NAMES"
	@echo ""
	@echo "2. Sample App:"
	@curl -s http://localhost:8080/health | jq -c || echo "❌ Sample app not responding"
	@echo ""
	@echo "3. AURA:"
	@curl -s http://localhost:8081/health | jq -c || echo "❌ AURA not responding"
	@echo ""
	@echo "4. Prometheus:"
	@curl -s http://localhost:9090/-/healthy || echo "❌ Prometheus not responding"
	@echo ""
	@echo "5. Database Metrics:"
	@docker exec aura-postgres psql -U aura -d aura_db -c "SELECT COUNT(*) FROM metrics;" -t 2>/dev/null || echo "❌ Database not responding"
	@echo ""

# Comprehensive test of all endpoints
test-all: health
	@echo "🧪 Testing all endpoints..."
	@echo ""
	@echo "Sample App Metrics:"
	@curl -s http://localhost:8080/metrics | grep -E "cpu_usage|memory_usage" | head -4
	@echo ""
	@echo "Prometheus Query:"
	@curl -s 'http://localhost:9090/api/v1/query?query=up' | jq -r '.data.result[] | "\(.metric.job): \(.value[1])"'
	@echo ""
	@echo "Recent Database Metrics:"
	@docker exec aura-postgres psql -U aura -d aura_db -c "SELECT service_name, metric_name, ROUND(metric_value::numeric, 2) as value FROM metrics ORDER BY timestamp DESC LIMIT 5;"
	@echo ""
	@echo "✅ All tests complete"

# Build Linux binaries for Docker
build-linux:
	@echo "🐧 Building Linux binaries..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/aura-linux ./cmd/aura/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sample-app-linux ./examples/sample-app/main.go
	@echo "✅ Linux binaries built"

# Setup Kubernetes client (kubectl, minikube)
kube-client-setup:
	@echo "🔍 Checking for kubectl and minikube..."
	@if ! command -v kubectl >/dev/null 2>&1; then \
	  echo '❌ kubectl not found. Please install kubectl: https://kubernetes.io/docs/tasks/tools/'; \
	  exit 1; \
	else \
	  echo '✓ kubectl found'; \
	fi
	@if ! command -v minikube >/dev/null 2>&1; then \
	  echo '❌ minikube not found. Please install minikube: https://minikube.sigs.k8s.io/docs/start/'; \
	  exit 1; \
	else \
	  echo '✓ minikube found'; \
	fi
	@echo "✅ Kubernetes client setup complete."
