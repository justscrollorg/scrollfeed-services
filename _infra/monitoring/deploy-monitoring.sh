#!/bin/bash

echo "Deploying ScrollFeed Monitoring Setup..."

# Apply ServiceMonitors
echo "Applying ServiceMonitors..."
kubectl apply -f /home/anurag/justscrolls/justscrl/scrollfeed-services/_infra/monitoring/servicemonitor.yaml

# Apply Grafana Dashboards ConfigMap
echo "Applying Grafana Dashboards..."
kubectl apply -f /home/anurag/justscrolls/justscrl/scrollfeed-services/_infra/monitoring/grafana-dashboards-configmap.yaml

# Update service labels for monitoring
echo "Updating service labels for monitoring..."

# Analytics service
kubectl patch service analytics-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"go"}}}'

# News service  
kubectl patch service news-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"go"}}}'

# Video service
kubectl patch service video-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"go"}}}'

# Memes service
kubectl patch service memes-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"go"}}}'

# Articles service (C#)
kubectl patch service articlessvc -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"dotnet"}}}'

# Jokes service (C#)
kubectl patch service jokes-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"dotnet"}}}'

# Articles service (Go)
kubectl patch service articles-service -n default -p '{"metadata":{"labels":{"monitoring":"true","framework":"go"}}}'

echo "Monitoring setup completed!"
echo ""
echo "Access Grafana at: http://localhost:3000"
echo "Default credentials: admin/admin"
echo ""
echo "ServiceMonitors created for automatic Prometheus scraping"
echo "Dashboards will be available in Grafana once services start exposing metrics"
echo ""
echo "Next steps:"
echo "1. Rebuild and redeploy your services with the new metrics code"
echo "2. Run: go mod tidy && go mod download (for Go services)" 
echo "3. Run: dotnet restore (for .NET services)"
echo "4. Check if metrics are exposed: curl http://service-ip/metrics"
