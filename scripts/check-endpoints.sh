#!/bin/bash

# Check AURA, Prometheus, Sample App, and Kubernetes pod metrics endpoints

set -e

AURA_API="http://localhost:8081"
PROMETHEUS_API="http://localhost:9090"
SAMPLE_APP_API="http://localhost:8080"

function check_endpoint() {
  local url="$1"
  local label="$2"
  echo -n "Checking $label ($url): "
  if curl -fsS --max-time 5 "$url" >/dev/null; then
    echo "✅ UP"
  else
    echo "❌ DOWN"
  fi
}

echo "\n🔎 Checking main service endpoints..."
check_endpoint "$AURA_API/health" "AURA /health"
check_endpoint "$AURA_API/ready" "AURA /ready"
check_endpoint "$SAMPLE_APP_API/health" "Sample App /health"
check_endpoint "$PROMETHEUS_API/-/healthy" "Prometheus /-/healthy"

# Check Kubernetes pods endpoint (should return JSON with pods info)
echo -n "Checking Kubernetes pods endpoint: "
if curl -fsS --max-time 5 "$AURA_API/api/v1/kubernetes/pods" | grep -q 'pods'; then
  echo "✅ PODS DATA RECEIVED"
else
  echo "❌ NO PODS DATA"
fi

# Check Kubernetes pod metrics endpoint (should return JSON with metrics)
echo -n "Checking Kubernetes pod metrics endpoint: "
PODS_RESPONSE=$(curl -fsS --max-time 5 "$AURA_API/api/v1/kubernetes/pods" 2>/dev/null)
POD_COUNT=$(echo "$PODS_RESPONSE" | grep -o '"count":[0-9]\+' | grep -o '[0-9]\+')

if [ -n "$POD_COUNT" ] && [ "$POD_COUNT" -gt 0 ]; then
  # Extract first pod name using grep and sed
  POD_NAME=$(echo "$PODS_RESPONSE" | grep -o '"name":"[^"]*"' | head -1 | sed 's/"name":"//;s/"$//')
  
  if [ -n "$POD_NAME" ]; then
    if curl -fsS --max-time 5 "$AURA_API/api/v1/kubernetes/pods/$POD_NAME/metrics" 2>/dev/null | grep -q '"'; then
      echo "✅ METRICS FOR POD \"$POD_NAME\" RECEIVED"
    else
      echo "⚠️  POD \"$POD_NAME\" FOUND BUT NO METRICS (pod may still be starting)"
    fi
  else
    echo "❌ NO POD NAME EXTRACTED"
  fi
else
  echo "⚠️  NO PODS RUNNING (deploy a pod: kubectl run test-nginx --image=nginx:alpine)"
fi

echo "\nAll endpoint checks complete."
