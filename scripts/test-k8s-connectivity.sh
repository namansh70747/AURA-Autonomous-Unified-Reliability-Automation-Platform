#!/bin/bash

###############################################################################
# Test Kubernetes Connectivity for AURA
# This script verifies that AURA can communicate with Kubernetes API
###############################################################################

set -e

echo "🧪 Testing Kubernetes Connectivity"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 1. Check Minikube Status
echo ""
echo "1️⃣ Checking Minikube Status..."
if minikube status | grep -q "apiserver: Running"; then
    echo "   ✅ Minikube API server is running"
else
    echo "   ❌ Minikube API server is not running"
    exit 1
fi

# 2. Check kubectl connectivity
echo ""
echo "2️⃣ Checking kubectl connectivity..."
if kubectl get nodes >/dev/null 2>&1; then
    NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
    echo "   ✅ kubectl connected to cluster: ${NODE_NAME}"
else
    echo "   ❌ kubectl cannot connect"
    exit 1
fi

# 3. Get API server details
echo ""
echo "3️⃣ Getting API Server Details..."
API_SERVER=$(kubectl cluster-info | grep "control plane" | awk '{print $NF}')
API_PORT=$(echo "$API_SERVER" | sed -n 's/.*:\([0-9]\+\).*/\1/p')
echo "   API Server: ${API_SERVER}"
echo "   API Port: ${API_PORT}"

# 4. Test connectivity from Docker
echo ""
echo "4️⃣ Testing Docker → host.docker.internal connectivity..."
if docker run --rm alpine sh -c "timeout 3 nc -zv host.docker.internal ${API_PORT}" 2>&1 | grep -q "succeeded\|open"; then
    echo "   ✅ Port ${API_PORT} is accessible from Docker"
else
    echo "   ⚠️  Port ${API_PORT} may not be accessible from Docker"
fi

# 5. Check Docker kubeconfig
echo ""
echo "5️⃣ Checking Docker kubeconfig..."
if [ -f ~/.kube_docker/config ]; then
    DOCKER_API=$(grep "server:" ~/.kube_docker/config | head -1 | awk '{print $2}')
    echo "   Docker kubeconfig server: ${DOCKER_API}"
    
    # Check if port matches
    DOCKER_PORT=$(echo "$DOCKER_API" | sed -n 's/.*:\([0-9]\+\).*/\1/p')
    if [ "$DOCKER_PORT" = "$API_PORT" ]; then
        echo "   ✅ Port matches current API server"
    else
        echo "   ⚠️  Port mismatch! Docker: ${DOCKER_PORT}, Current: ${API_PORT}"
        echo "      Run ./scripts/setup-kubeconfig.sh to update"
    fi
else
    echo "   ❌ Docker kubeconfig not found at ~/.kube_docker/config"
    exit 1
fi

# 6. Test AURA Kubernetes endpoint
echo ""
echo "6️⃣ Testing AURA Kubernetes Endpoint..."
if ! docker ps -q -f name=aura-main >/dev/null 2>&1; then
    echo "   ❌ AURA container is not running"
    echo "      Start it with: docker-compose up -d"
    exit 1
fi

K8S_RESPONSE=$(curl -s http://localhost:8081/api/v1/kubernetes/pods 2>&1)

if echo "$K8S_RESPONSE" | grep -q '"pods"'; then
    POD_COUNT=$(echo "$K8S_RESPONSE" | jq -r '.count' 2>/dev/null || echo "?")
    echo "   ✅ AURA can access Kubernetes API!"
    echo "      Pods in default namespace: ${POD_COUNT}"
    
    # Show pods if any
    if [ "$POD_COUNT" != "0" ]; then
        echo ""
        echo "   📋 Pods:"
        echo "$K8S_RESPONSE" | jq -r '.pods[] | "      • \(.name) (\(.phase))"' 2>/dev/null || true
    fi
else
    echo "   ❌ AURA cannot access Kubernetes API"
    ERROR_MSG=$(echo "$K8S_RESPONSE" | jq -r '.error' 2>/dev/null || echo "$K8S_RESPONSE" | head -c 200)
    echo "      Error: ${ERROR_MSG}"
    echo ""
    echo "   🔧 Troubleshooting:"
    echo "      1. Check AURA logs: docker logs aura-main --tail 20"
    echo "      2. Restart AURA: docker-compose restart aura"
    echo "      3. Regenerate kubeconfig: ./scripts/setup-kubeconfig.sh"
    exit 1
fi

# 7. Deploy test pod (optional)
echo ""
echo "7️⃣ Optional: Deploy test pod to verify monitoring..."
read -p "   Deploy test nginx pod? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "   Deploying test-nginx pod..."
    kubectl run test-nginx --image=nginx:alpine --restart=Never 2>/dev/null || echo "   (Pod may already exist)"
    sleep 3
    
    # Check if AURA can see it
    K8S_RESPONSE=$(curl -s http://localhost:8081/api/v1/kubernetes/pods 2>&1)
    POD_COUNT=$(echo "$K8S_RESPONSE" | jq -r '.count' 2>/dev/null || echo "0")
    
    if [ "$POD_COUNT" -gt "0" ]; then
        echo "   ✅ Test pod deployed and visible to AURA!"
        echo ""
        echo "   📋 Visible pods:"
        echo "$K8S_RESPONSE" | jq -r '.pods[] | "      • \(.name) (\(.namespace)) - \(.phase)"' 2>/dev/null || true
        echo ""
        echo "   Clean up with: kubectl delete pod test-nginx"
    else
        echo "   ⚠️  Pod deployed but not yet visible (may take a few seconds)"
    fi
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Kubernetes Connectivity Test Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 Quick Commands:"
echo "   • List pods via AURA: curl http://localhost:8081/api/v1/kubernetes/pods | jq"
echo "   • Get pod metrics: curl http://localhost:8081/api/v1/kubernetes/pods/POD_NAME/metrics | jq"
echo "   • Deploy test pod: kubectl run test-nginx --image=nginx:alpine"
echo "   • Watch AURA logs: docker logs -f aura-main"
echo ""
