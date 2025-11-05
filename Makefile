.PHONY: help build run clean docker-up docker-down test-all test-phase2 analyze health minikube-start minikube-stop k8s-setup

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

PROJECT_NAME := AURA
VERSION := 0.2.0
BUILD_DIR := bin
DOCKER_COMPOSE := docker-compose -f docker-compose.yml
KUBECONFIG_DOCKER := $(HOME)/.kube_docker/config

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
MAGENTA := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color
BOLD := \033[1m

# Icons
CHECK := ✅
CROSS := ❌
ARROW := ➜
ROCKET := 🚀
GEAR := ⚙️
K8S := ☸️

# ─────────────────────────────────────────────────────────────────────────────
# Help Command
# ─────────────────────────────────────────────────────────────────────────────

help:
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)$(GREEN)  $(PROJECT_NAME) - Autonomous Unified Reliability Automation$(NC)"
	@echo "$(BOLD)$(GREEN)                         Version $(VERSION)$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)🚀 Quick Start:$(NC)"
	@echo "  $(CYAN)make start-all$(NC)          - Complete setup with Minikube + Docker + AURA"
	@echo "  $(CYAN)make start$(NC)              - Start AURA (Docker only)"
	@echo "  $(CYAN)make run$(NC)                - Run AURA (services must be up)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)🔨 Build Commands:$(NC)"
	@echo "  make build              - Build all binaries for macOS"
	@echo "  make build-linux        - Build Linux binaries (for Docker)"
	@echo "  make build-sample-app   - Build sample-app binary"
	@echo "  make clean              - Clean build artifacts"
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(K8S) Kubernetes Commands:$(NC)"
	@echo "  $(CYAN)make minikube-start$(NC)     - Start Minikube cluster"
	@echo "  $(CYAN)make minikube-stop$(NC)      - Stop Minikube cluster"
	@echo "  $(CYAN)make minikube-status$(NC)    - Check Minikube status"
	@echo "  $(CYAN)make k8s-setup$(NC)          - Setup Kubernetes config for Docker"
	@echo "  make k8s-deploy         - Deploy AURA to Kubernetes"
	@echo "  make k8s-status         - Show Kubernetes pod status"
	@echo "  make k8s-clean          - Remove Kubernetes resources"
	@echo ""
	@echo "$(BOLD)$(YELLOW)🐳 Docker Commands:$(NC)"
	@echo "  make docker-up          - Start all Docker services"
	@echo "  make docker-down        - Stop all Docker services"
	@echo "  make docker-restart     - Restart all services"
	@echo "  make logs               - Show Docker logs"
	@echo "  make logs-aura          - Show AURA logs only"
	@echo ""
	@echo "$(BOLD)$(YELLOW)🧪 Testing Commands:$(NC)"
	@echo "  make test-all           - Test all endpoints"
	@echo "  make test-phase2        - Test Phase 2 pattern analysis"
	@echo "  make test-endpoints     - Test core endpoints"
	@echo "  make health             - System health check"
	@echo "  make analyze            - Run pattern analysis"
	@echo "  make diagnose           - Run full diagnosis"
	@echo ""
	@echo "$(BOLD)$(YELLOW)📊 Monitoring Commands:$(NC)"
	@echo "  make status             - Show service status"
	@echo "  make metrics            - Show current metrics"
	@echo "  make db-status          - Show database statistics"
	@echo "  make db-shell           - Open PostgreSQL shell"
	@echo ""
	@echo "$(BOLD)$(YELLOW)⚡ Performance Commands:$(NC)"
	@echo "  make benchmark          - Run performance benchmarks"
	@echo "  make load-test          - Run load test (50 requests)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)🧹 Cleanup Commands:$(NC)"
	@echo "  make stop-all           - Stop everything (Minikube + Docker)"
	@echo "  make reset              - Reset everything"
	@echo "  make prune              - Prune Docker resources"
	@echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Build Commands
# ─────────────────────────────────────────────────────────────────────────────

build:
	@echo "$(BLUE)🔨 Building AURA for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/aura ./cmd/aura/main.go
	@echo "$(GREEN)$(CHECK) Build complete: $(BUILD_DIR)/aura$(NC)"

build-linux:
	@echo "$(BLUE)🐧 Building Linux binaries for Docker...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/aura-linux ./cmd/aura/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/sample-app-linux ./examples/sample-app/main.go
	@echo "$(GREEN)$(CHECK) Linux binaries built$(NC)"
	@ls -lh $(BUILD_DIR)/*-linux

build-sample-app:
	@echo "$(BLUE)🔨 Building sample-app for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/sample-app-mac ./examples/sample-app/main.go
	@echo "$(GREEN)$(CHECK) Sample-app built: $(BUILD_DIR)/sample-app-mac$(NC)"

clean:
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f /tmp/sample-app.log /tmp/aura*.go
	@echo "$(GREEN)$(CHECK) Clean complete$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Minikube Commands
# ─────────────────────────────────────────────────────────────────────────────

minikube-check:
	@echo "$(BLUE)$(K8S) Checking Minikube installation...$(NC)"
	@command -v minikube >/dev/null 2>&1 || { \
	    echo "$(RED)$(CROSS) Minikube is not installed$(NC)"; \
	    echo "$(YELLOW)$(ARROW) Install: brew install minikube$(NC)"; \
	    exit 1; \
	}
	@command -v kubectl >/dev/null 2>&1 || { \
	    echo "$(RED)$(CROSS) kubectl is not installed$(NC)"; \
	    echo "$(YELLOW)$(ARROW) Install: brew install kubectl$(NC)"; \
	    exit 1; \
	}
	@echo "$(GREEN)$(CHECK) Minikube and kubectl are installed$(NC)"

minikube-start: minikube-check
	@echo "$(BLUE)$(K8S) Starting Minikube cluster...$(NC)"
	@if minikube status | grep -q "Running"; then \
	    echo "$(GREEN)$(CHECK) Minikube is already running$(NC)"; \
	else \
	    echo "$(YELLOW)⏳ Starting Minikube (this may take 1-2 minutes)...$(NC)"; \
	    minikube start --driver=docker --cpus=2 --memory=4096; \
	    echo "$(GREEN)$(CHECK) Minikube started successfully$(NC)"; \
	fi
	@echo ""
	@echo "$(CYAN)$(ARROW) Minikube IP: $$(minikube ip)$(NC)"
	@echo "$(CYAN)$(ARROW) Kubernetes version: $$(kubectl version --short 2>/dev/null | grep Server | awk '{print $$3}')$(NC)"

minikube-stop:
	@echo "$(BLUE)$(K8S) Stopping Minikube cluster...$(NC)"
	@minikube stop || true
	@echo "$(GREEN)$(CHECK) Minikube stopped$(NC)"

minikube-status:
	@echo "$(BLUE)$(K8S) Minikube Status$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@minikube status || echo "$(YELLOW)Minikube is not running$(NC)"
	@echo ""
	@echo "$(YELLOW)Kubernetes Nodes:$(NC)"
	@kubectl get nodes 2>/dev/null || echo "$(RED)Cannot connect to Kubernetes$(NC)"

minikube-delete:
	@echo "$(BLUE)$(K8S) Deleting Minikube cluster...$(NC)"
	@echo "$(RED)⚠️  This will delete all data in Minikube!$(NC)"
	@read -p "Are you sure? (y/N): " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
	    minikube delete; \
	    echo "$(GREEN)$(CHECK) Minikube cluster deleted$(NC)"; \
	else \
	    echo "$(YELLOW)Cancelled$(NC)"; \
	fi

# ─────────────────────────────────────────────────────────────────────────────
# Kubernetes Setup
# ─────────────────────────────────────────────────────────────────────────────

k8s-setup: minikube-start
	@echo "$(BLUE)$(GEAR) Setting up Kubernetes config for Docker...$(NC)"
	@echo ""
	@mkdir -p $(HOME)/.kube_docker
	@if [ -f $(HOME)/.kube/config ]; then \
	    echo "$(CYAN)$(ARROW) Copying kubeconfig...$(NC)"; \
	    cp $(HOME)/.kube/config $(KUBECONFIG_DOCKER);s \
	    echo "$(CYAN)$(ARROW) Modifying for Docker compatibility...$(NC)"; \
	    if [[ "$$OSTYPE" == "darwin"* ]]; then \
	        sed -i '' 's/127\.0\.0\.1/host.docker.internal/g' $(KUBECONFIG_DOCKER); \
	        sed -i '' 's/localhost/host.docker.internal/g' $(KUBECONFIG_DOCKER); \
	        sed -i '' 's/certificate-authority:.*/insecure-skip-tls-verify: true/g' $(KUBECONFIG_DOCKER); \
	    else \
	        sed -i 's/127\.0\.0\.1/host.docker.internal/g' $(KUBECONFIG_DOCKER); \
	        sed -i 's/localhost/host.docker.internal/g' $(KUBECONFIG_DOCKER); \
	        sed -i 's/certificate-authority:.*/insecure-skip-tls-verify: true/g' $(KUBECONFIG_DOCKER); \
	    fi; \
	    echo "$(GREEN)$(CHECK) Kubeconfig configured: $(KUBECONFIG_DOCKER)$(NC)"; \
	else \
	    echo "$(RED)$(CROSS) Kubeconfig not found at $(HOME)/.kube/config$(NC)"; \
	    exit 1; \
	fi
	@echo ""
	@echo "$(CYAN)$(ARROW) Testing Kubernetes connectivity...$(NC)"
	@if kubectl --kubeconfig=$(KUBECONFIG_DOCKER) get nodes >/dev/null 2>&1; then \
	    echo "$(GREEN)$(CHECK) Kubernetes is accessible$(NC)"; \
	else \
	    echo "$(YELLOW)⚠️  Kubernetes not accessible from host (will retry from Docker)$(NC)"; \
	fi

k8s-verify:
	@echo "$(BLUE)$(K8S) Verifying Kubernetes Setup$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Minikube Status:$(NC)"
	@minikube status | head -5 || echo "$(RED)$(CROSS) Minikube not running$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Kubernetes Nodes:$(NC)"
	@kubectl get nodes || echo "$(RED)$(CROSS) Cannot connect$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Kubeconfig Location:$(NC)"
	@echo "   Host:   $(HOME)/.kube/config"
	@echo "   Docker: $(KUBECONFIG_DOCKER)"
	@ls -lh $(KUBECONFIG_DOCKER) 2>/dev/null || echo "   $(RED)Not found$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Kubernetes Deployment
# ─────────────────────────────────────────────────────────────────────────────

k8s-deploy:
	@echo "$(BLUE)$(K8S) Deploying to Kubernetes...$(NC)"
	@kubectl apply -f deployments/k8s/
	@echo "$(GREEN)$(CHECK) Kubernetes resources deployed$(NC)"

k8s-status:
	@echo "$(BLUE)$(K8S) Kubernetes Status$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(YELLOW)Pods:$(NC)"
	@kubectl get pods --all-namespaces | head -10
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@kubectl get services --all-namespaces | head -10

k8s-clean:
	@echo "$(BLUE)🧹 Cleaning Kubernetes resources...$(NC)"
	@kubectl delete -f deployments/k8s/ --ignore-not-found=true
	@echo "$(GREEN)$(CHECK) Kubernetes resources removed$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Docker Commands
# ─────────────────────────────────────────────────────────────────────────────

docker-up:
	@echo "$(BLUE)🐳 Starting Docker services...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "⏳ Waiting for services..."
	@sleep 10
	@echo "$(GREEN)$(CHECK) Services started:$(NC)"
	@echo "   $(ROCKET) AURA API:       http://localhost:8081"
	@echo "   📊 Prometheus:     http://localhost:9090"
	@echo "   💾 PostgreSQL:     localhost:5432"
	@echo "   🌐 Sample App:     http://localhost:8080"

docker-down:
	@echo "$(BLUE)🛑 Stopping Docker services...$(NC)"
	$(DOCKER_COMPOSE) down --remove-orphans
	@pkill -f "sample-app-mac" 2>/dev/null || true
	@echo "$(GREEN)$(CHECK) Services stopped$(NC)"

docker-restart:
	@echo "$(BLUE)🔄 Restarting Docker services...$(NC)"
	@$(MAKE) docker-down
	@sleep 2
	@$(MAKE) docker-up

logs:
	@echo "$(BLUE)📜 Docker logs:$(NC)"
	$(DOCKER_COMPOSE) logs --tail=50 --follow

logs-aura:
	@echo "$(BLUE)📜 AURA logs:$(NC)"
	docker logs aura-main --tail=50 --follow

# ─────────────────────────────────────────────────────────────────────────────
# Complete Setup Commands
# ─────────────────────────────────────────────────────────────────────────────

start-all: clean minikube-start k8s-setup build-linux docker-up
	@echo ""
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)$(GREEN)🎉 AURA Platform Started Successfully!$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Access Points:$(NC)"
	@echo "  $(ROCKET) AURA API:       $(CYAN)http://localhost:8081$(NC)"
	@echo "  📊 Prometheus:     $(CYAN)http://localhost:9090$(NC)"
	@echo "  💾 PostgreSQL:     $(CYAN)localhost:5432$(NC)"
	@echo "  🌐 Sample App:     $(CYAN)http://localhost:8080$(NC)"
	@echo "  $(K8S) Minikube IP:    $(CYAN)$$(minikube ip)$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Quick Commands:$(NC)"
	@echo "  $(CYAN)make health$(NC)        - System health check"
	@echo "  $(CYAN)make test-phase2$(NC)   - Test Phase 2 endpoints"
	@echo "  $(CYAN)make analyze$(NC)       - Run pattern analysis"
	@echo "  $(CYAN)make k8s-status$(NC)    - Check Kubernetes pods"
	@echo "  $(CYAN)make stop-all$(NC)      - Stop everything"
	@echo ""

start: k8s-setup build-linux docker-up
	@echo "$(GREEN)$(CHECK) AURA platform is running!$(NC)"
	@echo "$(GREEN)$(CHECK) All services initialized$(NC)"

init: k8s-setup build-linux docker-up
	@echo "$(GREEN)🎉 AURA initialized successfully!$(NC)"
	@echo "Run 'make run' to start AURA"

# ─────────────────────────────────────────────────────────────────────────────
# Run Commands
# ─────────────────────────────────────────────────────────────────────────────

run:
	@echo "$(BLUE)$(ROCKET) Starting AURA...$(NC)"
	@if [ ! -f "$(BUILD_DIR)/aura-linux" ]; then \
	    echo "$(RED)$(CROSS) Binary not found. Run 'make build-linux' first$(NC)"; \
	    exit 1; \
	fi
	@export ENVIRONMENT=development AURA_CONFIG_PATH=configs/aura.yaml && \
	    ./$(BUILD_DIR)/aura-linux

# ─────────────────────────────────────────────────────────────────────────────
# Testing Commands
# ─────────────────────────────────────────────────────────────────────────────

test-endpoints:
	@echo "$(BLUE)🧪 Testing AURA API Endpoints...$(NC)"
	@echo ""
	@echo "$(YELLOW)Core Endpoints:$(NC)"
	@curl -sf http://localhost:8081/health | jq . && echo "$(GREEN)$(CHECK) /health$(NC)" || echo "$(RED)$(CROSS) /health$(NC)"
	@curl -sf http://localhost:8081/ready | jq . && echo "$(GREEN)$(CHECK) /ready$(NC)" || echo "$(RED)$(CROSS) /ready$(NC)"
	@curl -sf http://localhost:8081/api/v1/status | jq . && echo "$(GREEN)$(CHECK) /status$(NC)" || echo "$(RED)$(CROSS) /status$(NC)"
	@echo ""
	@echo "$(YELLOW)Metrics Endpoints:$(NC)"
	@curl -sf http://localhost:8081/api/v1/metrics/sample-app | jq -c '.metrics' && echo "$(GREEN)$(CHECK) /metrics/sample-app$(NC)" || echo "$(RED)$(CROSS) /metrics/sample-app$(NC)"
	@curl -sf http://localhost:8081/api/v1/metrics/services | jq -c '.services' && echo "$(GREEN)$(CHECK) /metrics/services$(NC)" || echo "$(RED)$(CROSS) /metrics/services$(NC)"

test-phase2:
	@echo "$(BLUE)🧪 Testing Phase 2: Pattern Analysis$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Testing /analyze/:service endpoint$(NC)"
	@curl -sf http://localhost:8081/api/v1/analyze/sample-app | jq '{service: .service, problem: .diagnosis.problem, confidence: .diagnosis.confidence, severity: .diagnosis.severity}' && echo "$(GREEN)$(CHECK) Single service analysis$(NC)" || echo "$(RED)$(CROSS) Single service analysis failed$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Testing /analyze/all endpoint$(NC)"
	@curl -sf http://localhost:8081/api/v1/analyze/all | jq '{total_services: .total_services, services: .services}' && echo "$(GREEN)$(CHECK) All services analysis$(NC)" || echo "$(RED)$(CROSS) All services analysis failed$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Checking diagnosis in database$(NC)"
	@docker exec aura-postgres psql -U aura -d aura_db -c "SELECT COUNT(*) FROM diagnoses;" -t | xargs echo "Diagnoses in DB:"
	@echo ""
	@echo "$(GREEN)$(CHECK) Phase 2 testing complete$(NC)"

test-k8s:
	@echo "$(BLUE)🧪 Testing Kubernetes Integration$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Testing Kubernetes API endpoint$(NC)"
	@curl -sf http://localhost:8081/api/v1/kubernetes/pods | jq '{count: .count}' && echo "$(GREEN)$(CHECK) Kubernetes endpoint working$(NC)" || echo "$(RED)$(CROSS) Kubernetes endpoint failed$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Minikube connectivity$(NC)"
	@kubectl get nodes && echo "$(GREEN)$(CHECK) Minikube accessible$(NC)" || echo "$(RED)$(CROSS) Minikube not accessible$(NC)"

test-all: health test-endpoints test-phase2 test-k8s
	@echo ""
	@echo "$(GREEN)$(CHECK) All tests complete$(NC)"

analyze:
	@echo "$(BLUE)🔍 Running pattern analysis on all services...$(NC)"
	@curl -s http://localhost:8081/api/v1/analyze/all | jq

diagnose:
	@echo "$(BLUE)🩺 Running comprehensive diagnosis...$(NC)"
	@echo ""
	@$(MAKE) analyze
	@echo ""
	@echo "$(YELLOW)Recent Diagnoses:$(NC)"
	@docker exec aura-postgres psql -U aura -d aura_db -c "SELECT service_name, problem_type, ROUND(confidence::numeric, 2) as confidence, severity, TO_CHAR(timestamp, 'YYYY-MM-DD HH24:MI:SS') as time FROM diagnoses ORDER BY timestamp DESC LIMIT 5;"

# ─────────────────────────────────────────────────────────────────────────────
# Health & Monitoring Commands
# ─────────────────────────────────────────────────────────────────────────────

health:
	@echo "$(BLUE)🏥 System Health Check$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Minikube:$(NC)"
	@minikube status | head -3 && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(RED)$(CROSS) DOWN$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Docker Services:$(NC)"
	@docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "aura|sample|postgres|prometheus" && echo "" || echo "$(RED)$(CROSS) No services running$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Sample App:$(NC)"
	@curl -sf http://localhost:8080/health | jq -c && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(RED)$(CROSS) DOWN$(NC)"
	@echo ""
	@echo "$(YELLOW)4. AURA API:$(NC)"
	@curl -sf http://localhost:8081/health | jq -c && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(RED)$(CROSS) DOWN$(NC)"
	@echo ""
	@echo "$(YELLOW)5. Prometheus:$(NC)"
	@curl -sf http://localhost:9090/-/healthy && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(RED)$(CROSS) DOWN$(NC)"
	@echo ""
	@echo "$(YELLOW)6. PostgreSQL:$(NC)"
	@docker exec aura-postgres pg_isready -U aura && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(RED)$(CROSS) DOWN$(NC)"
	@echo ""
	@echo "$(YELLOW)7. Kubernetes:$(NC)"
	@curl -sf http://localhost:8081/api/v1/kubernetes/pods | jq -c '.count' | xargs -I {} echo "Pods: {}" && echo "$(GREEN)$(CHECK) UP$(NC)" || echo "$(YELLOW)⚠️  K8s not available$(NC)"

status:
	@echo "$(BLUE)📊 Service Status$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@curl -sf http://localhost:8081/api/v1/status | jq
	@echo ""
	@echo "$(YELLOW)Docker Containers:$(NC)"
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "$(YELLOW)Kubernetes Pods:$(NC)"
	@kubectl get pods 2>/dev/null || echo "Kubernetes not available"

metrics:
	@echo "$(BLUE)📈 Current Metrics$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@curl -sf http://localhost:8081/api/v1/metrics/sample-app | jq '.metrics'

# ─────────────────────────────────────────────────────────────────────────────
# Database Commands
# ─────────────────────────────────────────────────────────────────────────────

db-status:
	@echo "$(BLUE)💾 Database Status$(NC)"
	@echo "$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@docker exec aura-postgres psql -U aura -d aura_db <<< "\
	    SELECT 'Metrics' as table_name, COUNT(*) as count FROM metrics \
	    UNION ALL \
	    SELECT 'Diagnoses', COUNT(*) FROM diagnoses \
	    UNION ALL \
	    SELECT 'Decisions', COUNT(*) FROM decisions \
	    UNION ALL \
	    SELECT 'Events', COUNT(*) FROM events;"

db-shell:
	@echo "$(BLUE)💾 Opening PostgreSQL shell...$(NC)"
	@docker exec -it aura-postgres psql -U aura -d aura_db

# ─────────────────────────────────────────────────────────────────────────────
# Performance Commands
# ─────────────────────────────────────────────────────────────────────────────

benchmark:
	@echo "$(BLUE)⚡ Running performance benchmarks...$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Health endpoint:$(NC)"
	@time curl -sf http://localhost:8081/health > /dev/null
	@echo ""
	@echo "$(YELLOW)2. Metrics endpoint:$(NC)"
	@time curl -sf http://localhost:8081/api/v1/metrics/sample-app > /dev/null
	@echo ""
	@echo "$(YELLOW)3. Analysis endpoint:$(NC)"
	@time curl -sf http://localhost:8081/api/v1/analyze/sample-app > /dev/null

load-test:
	@echo "$(BLUE)🏋️  Running load test (50 requests)...$(NC)"
	@for i in {1..50}; do \
	    curl -sf http://localhost:8081/health > /dev/null & \
	done
	@wait
	@echo "$(GREEN)$(CHECK) Load test complete$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Development Commands
# ─────────────────────────────────────────────────────────────────────────────

dev: build-linux docker-up
	@echo "$(GREEN)🔧 Development environment ready$(NC)"
	@echo "Run 'make run' in another terminal"

watch:
	@echo "$(BLUE)👀 Watching logs...$(NC)"
	@$(MAKE) logs

tail:
	@tail -f logs/aura.log /tmp/sample-app.log

# ─────────────────────────────────────────────────────────────────────────────
# Cleanup Commands
# ─────────────────────────────────────────────────────────────────────────────

stop-all:
	@echo "$(BLUE)🛑 Stopping all services...$(NC)"
	@$(MAKE) docker-down
	@$(MAKE) minikube-stop
	@echo "$(GREEN)$(CHECK) All services stopped$(NC)"

reset: clean docker-down
	@echo "$(BLUE)🔄 Resetting everything...$(NC)"
	@docker volume rm aura-postgres-data 2>/dev/null || true
	@rm -rf logs/*.log /tmp/sample-app.log
	@echo "$(GREEN)$(CHECK) Reset complete$(NC)"

prune:
	@echo "$(BLUE)🧹 Pruning Docker resources...$(NC)"
	@docker system prune -af --volumes
	@echo "$(GREEN)$(CHECK) Prune complete$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Advanced Commands
# ─────────────────────────────────────────────────────────────────────────────

full-reset: clean docker-down minikube-delete
	@echo "$(BLUE)🔄 Full system reset...$(NC)"
	@docker volume rm aura-postgres-data 2>/dev/null || true
	@rm -rf logs/*.log /tmp/sample-app.log
	@rm -rf $(HOME)/.kube_docker
	@echo "$(GREEN)$(CHECK) Full reset complete$(NC)"

quick-start: start-all test-all
	@echo ""
	@echo "$(BOLD)$(GREEN)🎊 AURA is ready and tested!$(NC)"

# ─────────────────────────────────────────────────────────────────────────────
# Help Aliases
# ─────────────────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help
