# News Service Architecture - Event-Driven Design

## Overview

This project implements an event-driven news fetching and serving system using NATS JetStream for messaging, separating concerns between data fetching and API serving.

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   News Fetcher  │───▶│   NATS JetStream │───▶│  News Service   │
│    Service      │    │   (Events/Jobs)  │    │   (API Only)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                        │
         │                        ▼                        │
         │              ┌──────────────────┐               │
         └─────────────▶│   MongoDB        │◀──────────────┘
                        │   (Articles)     │
                        └──────────────────┘
```

## Services

### 1. News Fetcher Service (`news-fetcher-service/`)

**Responsibilities:**
- Fetches articles from GNews API
- Handles rate limiting and retries
- Stores articles in MongoDB
- Processes fetch requests from NATS
- Publishes fetch results to NATS

**Key Features:**
- **Configurable fetch intervals** (default: 4 hours)
- **Worker-based processing** (default: 3 workers)
- **Intelligent rate limiting** (1 second between API calls)
- **Retry logic** with exponential backoff
- **Graceful error handling**

**Environment Variables:**
```bash
MONGO_URI=mongodb://localhost:27017
NATS_URL=nats://localhost:4222
GNEWS_API_KEY=your_api_key_here
FETCH_INTERVAL=4h
RATE_LIMIT=1s
MAX_RETRIES=3
RETRY_DELAY=30s
WORKER_COUNT=3
```

### 2. News Service (`news-service/`)

**Responsibilities:**
- Serves news articles via REST API
- Allows manual triggering of news fetches
- Monitors fetch results
- Provides filtering by region

**API Endpoints:**
```http
GET /news-api/news?region=us           # Get news articles
POST /news-api/fetch/us?priority=high  # Trigger fetch for specific region
POST /news-api/fetch-all?priority=high # Trigger fetch for all regions
```

## NATS Subjects

### Streams
- **NEWS_FETCH**: Handles fetch requests and results
  - Retention: Work queue policy
  - Storage: File storage
  - Max age: 24 hours

### Subjects
- `news.fetch.request`: Fetch requests
- `news.fetch.result`: Fetch results/status

## Benefits of This Architecture

### 1. **Scalability**
- Independent scaling of fetcher and API services
- Horizontal scaling with multiple workers
- Load balancing across service instances

### 2. **Reliability**
- Message persistence with NATS JetStream
- Retry logic for failed fetches
- Graceful degradation on API failures

### 3. **Rate Limit Compliance**
- Centralized rate limiting in fetcher service
- Configurable delays between API calls
- Prevents API quota violations

### 4. **Flexibility**
- Manual fetch triggering via API
- Priority-based processing
- Configurable fetch intervals

### 5. **Monitoring**
- Real-time fetch status monitoring
- Detailed logging for debugging
- Health checks for all services

## Rate Limiting Strategy

### Current Implementation
1. **1-second delay** between API pages
2. **Maximum 4 pages** per region per fetch
3. **3 workers** processing requests concurrently
4. **4-hour intervals** for automatic fetches

### API Rate Limits (GNews)
- **100 requests/day** for free tier
- **Daily usage calculation**:
  - 3 regions × 4 pages × 6 fetches/day = 72 requests/day
  - Leaves 28 requests for manual triggers

### Optimization Strategies
1. **Smart caching**: Avoid re-fetching recent articles
2. **Conditional fetching**: Check for new content before full fetch
3. **Priority queues**: High-priority manual requests first
4. **Back-pressure**: Slow down during high load

## Deployment

### Local Development
```bash
# Start NATS
docker run -p 4222:4222 nats:latest

# Start MongoDB
docker run -p 27017:27017 mongo:latest

# Start News Fetcher Service
cd news-fetcher-service
go run cmd/main.go

# Start News Service
cd news-service
go run cmd/main.go
```

### Kubernetes
```bash
# Deploy NATS (already configured in _infra/terraform/modules/nats/)
kubectl apply -f _infra/terraform/modules/nats/

# Deploy services
kubectl apply -f _infra/news-fetcher-service/deployment.yaml
kubectl apply -f _infra/news-service/news-service.yaml
```

## Monitoring and Observability

### Logs
- Structured JSON logging
- Request/response tracing
- Error tracking with context

### Metrics (Future Enhancement)
- Fetch success/failure rates
- API response times
- Queue depth monitoring
- Article count per region

### Health Checks
- Database connectivity
- NATS connectivity
- Service readiness probes

## Configuration Management

### Environment-based Configuration
All services use environment variables for configuration, allowing easy deployment across different environments.

### Secret Management
- API keys stored in Kubernetes secrets
- Database credentials managed securely
- No sensitive data in code/images

## Future Enhancements

### 1. Content Deduplication
- Hash-based article deduplication
- Prevent storing duplicate articles
- Reduce storage requirements

### 2. Advanced Scheduling
- Different intervals per region
- Peak/off-peak scheduling
- User activity-based fetching

### 3. Content Analysis
- Sentiment analysis
- Topic categorization
- Trending article detection

### 4. Performance Optimization
- Redis caching layer
- CDN for article images
- Database indexing optimization

### 5. Monitoring Dashboard
- Real-time fetch status
- API usage metrics
- System health overview

## Best Practices Implemented

1. **Separation of Concerns**: Fetching and serving are separate services
2. **Event-Driven Architecture**: Loose coupling via message queues
3. **Graceful Error Handling**: Continues operation during partial failures
4. **Resource Management**: Configurable limits and resource allocation
5. **Observability**: Comprehensive logging and health checks
6. **Security**: Environment-based secrets and minimal privileges

This architecture provides a robust, scalable, and maintainable solution for news aggregation while respecting API rate limits and providing excellent user experience.
