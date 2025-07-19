#!/bin/bash

echo "=== News Service Architecture Test ==="
echo "This script demonstrates the new event-driven architecture"
echo ""

# Check if services are built
if [ ! -f "news-service/news-service.exe" ]; then
    echo "❌ News service not built. Run: cd news-service && go build -o news-service.exe ./cmd"
    exit 1
fi

if [ ! -f "news-fetcher-service/news-fetcher.exe" ]; then
    echo "❌ News fetcher service not built. Run: cd news-fetcher-service && go build -o news-fetcher.exe ./cmd"
    exit 1
fi

echo "✅ Both services are built"
echo ""

echo "=== Testing API Endpoints ==="
echo "Start services manually with proper environment variables:"
echo ""
echo "Terminal 1 - Start NATS:"
echo "docker run -p 4222:4222 -p 8222:8222 nats:latest -js"
echo ""
echo "Terminal 2 - Start MongoDB:"
echo "docker run -p 27017:27017 mongo:latest"
echo ""
echo "Terminal 3 - Start News Service:"
echo "cd news-service"
echo "set MONGO_URI=mongodb://localhost:27017"
echo "set NATS_URL=nats://localhost:4222"
echo "./news-service.exe"
echo ""
echo "Terminal 4 - Start News Fetcher Service:"
echo "cd news-fetcher-service"
echo "set MONGO_URI=mongodb://localhost:27017"
echo "set NATS_URL=nats://localhost:4222"
echo "set GNEWS_API_KEY=your_api_key_here"
echo "./news-fetcher.exe"
echo ""
echo "Then test with these API calls:"
echo ""
echo "# Get all news articles"
echo "curl http://localhost:8080/news-api/news"
echo ""
echo "# Get news for specific region"
echo "curl http://localhost:8080/news-api/news?region=us"
echo ""
echo "# Trigger manual fetch for US"
echo "curl -X POST http://localhost:8080/news-api/fetch/us?priority=high"
echo ""
echo "# Trigger fetch for all regions"
echo "curl -X POST http://localhost:8080/news-api/fetch-all?priority=high"
