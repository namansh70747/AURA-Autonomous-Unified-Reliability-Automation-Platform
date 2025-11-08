#!/bin/bash

# Test script to demonstrate multiple problem detection

echo "=== Testing Multiple Problem Detection ==="
echo ""

echo "📊 Current Analysis:"
curl -s "http://localhost:8081/api/v1/analyze/sample-app" | jq '{
  service: .service,
  problem: .diagnosis.problem,
  confidence: .diagnosis.confidence,
  severity: .diagnosis.severity,
  multiple_problems: .diagnosis.multiple_problems,
  high_confidence_count: .diagnosis.high_confidence_count
}'

echo ""
echo "🔍 All Detection Confidence Levels:"
curl -s "http://localhost:8081/api/v1/analyze/sample-app" | jq '.diagnosis.all_detections | map({
  type: .type,
  detected: .detected,
  confidence: .confidence,
  severity: .severity
})'

echo ""
echo "💾 Diagnoses Saved in Last Minute (all >80% confidence):"
docker exec aura-postgres psql -U aura -d aura_db -c "
SELECT 
    problem_type,
    ROUND(confidence::numeric, 2) as confidence,
    severity,
    TO_CHAR(timestamp, 'HH24:MI:SS') as time
FROM diagnoses 
WHERE timestamp > NOW() - INTERVAL '1 minute'
ORDER BY timestamp DESC, confidence DESC;
"

echo ""
echo "📈 All Problems Ever Detected (Grouped):"
docker exec aura-postgres psql -U aura -d aura_db -c "
SELECT 
    problem_type,
    COUNT(*) as occurrences,
    ROUND(AVG(confidence)::numeric, 2) as avg_confidence,
    MAX(severity) as max_severity
FROM diagnoses 
GROUP BY problem_type
ORDER BY occurrences DESC;
"

echo ""
echo "✅ Test Complete"
