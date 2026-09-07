# Moly Deployment & Operations Guide

**Date**: September 6, 2026  
**Status**: Operations Reference  
**Version**: 1.0

---

## Overview

Complete guide for deploying Moly backend, managing infrastructure, and scaling.

---

## Prerequisites

- **Go 1.21+** — Backend runtime
- **PostgreSQL 14+** — Database
- **Docker** (optional) — Containerization
- **Claude API key** — For LLM access
- **Anthropic API** — For extended thinking

---

## Backend Setup

### 1. Clone Repository

```bash
git clone https://github.com/yourusername/moly-backend.git
cd moly-backend
```

### 2. Install Dependencies

```bash
go mod download
go mod tidy
```

### 3. Environment Configuration

Create `.env` file:

```bash
# Database
DATABASE_URL=postgres://user:password@localhost:5432/moly
DB_MAX_CONNECTIONS=20
DB_IDLE_TIMEOUT=5m

# API Keys
CLAUDE_API_KEY=sk-ant-...
ANTHROPIC_API_KEY=sk-ant-...

# Server
PORT=11436
HOST=localhost
ENVIRONMENT=development
LOG_LEVEL=info

# Agent Configuration
AGENT_MODEL=claude-opus-5
AGENT_TEMPERATURE=0.7
AGENT_MAX_TOKENS=2000

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60 # seconds

# CORS
ALLOWED_ORIGINS=chrome-extension://*

# Monitoring
SENTRY_DSN=
DATADOG_API_KEY=
```

### 4. Database Setup

```bash
# Create database
createdb moly

# Run migrations
go run cmd/migrate/main.go up

# Verify
psql moly -c "SELECT table_name FROM information_schema.tables;"
```

### 5. Start Backend

```bash
# Development
go run cmd/server/main.go

# Production
go build -o moly-backend cmd/server/main.go
./moly-backend
```

**Expected output**:
```
[INFO] Moly backend starting...
[INFO] Database connected
[INFO] Agents initialized
[INFO] Server listening on :11436
```

---

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o moly-backend cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates postgresql-client
WORKDIR /root/

COPY --from=builder /app/moly-backend .
EXPOSE 11436

CMD ["./moly-backend"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  db:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: moly
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  backend:
    build: .
    depends_on:
      - db
    environment:
      DATABASE_URL: postgres://postgres:postgres@db:5432/moly
      CLAUDE_API_KEY: ${CLAUDE_API_KEY}
      PORT: 11436
    ports:
      - "11436:11436"
    volumes:
      - ./logs:/root/logs

volumes:
  postgres_data:
```

### Run

```bash
docker-compose up -d
docker-compose logs -f backend
```

---

## Extension Deployment

### 1. Build Extension

```bash
cd moly-extension
npm install
npm run build
```

**Output**: `dist/` directory with bundled extension

### 2. Load in Chrome

- Open `chrome://extensions`
- Enable "Developer mode"
- Click "Load unpacked"
- Select `dist/` directory

### 3. Package for Chrome Web Store

```bash
# Create zip
zip -r moly-extension.zip dist/

# Upload to Chrome Web Store
# https://chrome.google.com/webstore/devconsole
```

---

## Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: moly-backend
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: moly-backend
  template:
    metadata:
      labels:
        app: moly-backend
    spec:
      containers:
      - name: backend
        image: moly-backend:1.0.0
        ports:
        - containerPort: 11436
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: moly-secrets
              key: database-url
        - name: CLAUDE_API_KEY
          valueFrom:
            secretKeyRef:
              name: moly-secrets
              key: claude-api-key
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /api/status
            port: 11436
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/status
            port: 11436
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: moly-backend
  namespace: production
spec:
  selector:
    app: moly-backend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 11436
  type: LoadBalancer

---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: moly-backend-pdb
  namespace: production
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: moly-backend
```

### Deploy

```bash
kubectl apply -f deployment.yaml
kubectl get pods -n production
kubectl logs -f deployment/moly-backend -n production
```

---

## Scaling Configuration

### Agent Pool

```go
// agents/pool.go
type AgentPool struct {
  conversationAgents int = 10
  learningAgents int = 5
  contextManagers int = 5
  riskMonitors int = 3
  
  // Queue for overflow
  queue chan *AgentTask
}

// Config from environment
CONVERSATION_AGENTS=10
LEARNING_AGENTS=5
CONTEXT_MANAGERS=5
RISK_MONITORS=3
QUEUE_SIZE=1000
```

### Database Connection Pool

```go
// database/pool.go
type Config struct {
  MaxConnections int = 20 // 50 for production
  MinConnections int = 5
  MaxIdleTime time.Duration = 5 * time.Minute
  ConnectionTimeout time.Duration = 30 * time.Second
}

// PostgreSQL pgbouncer for connection pooling
```

### Load Balancing

```nginx
# nginx.conf
upstream moly_backend {
  least_conn; # Load balance by least connections
  server backend-1:11436;
  server backend-2:11436;
  server backend-3:11436;
}

server {
  listen 80;
  server_name api.moly.dev;
  
  location / {
    proxy_pass http://moly_backend;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_buffering off;
  }
}
```

---

## Monitoring & Alerting

### Prometheus Metrics

```go
// metrics/metrics.go
var (
  apiRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
      Name: "moly_api_requests_total",
      Help: "Total API requests",
    },
    []string{"method", "endpoint", "status"},
  )
  
  apiDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
      Name: "moly_api_duration_seconds",
      Help: "API request duration",
      Buckets: []float64{.1, .5, 1, 2, 5, 10},
    },
    []string{"method", "endpoint"},
  )
  
  agentProcessingTime = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
      Name: "moly_agent_processing_seconds",
      Help: "Agent processing time",
    },
    []string{"agent_type"},
  )
)
```

### Grafana Dashboard

Monitor:
- API request rate and latency
- Agent processing times
- Database connection pool usage
- Error rates by endpoint
- Cache hit rates
- Queue depth (if using message queue)

### Alerts

```yaml
# alerts.yaml
groups:
- name: moly
  rules:
  - alert: HighErrorRate
    expr: rate(moly_api_requests_total{status="5xx"}[5m]) > 0.05
    for: 5m
    annotations:
      summary: "High error rate detected"
  
  - alert: SlowAgent
    expr: histogram_quantile(0.99, moly_agent_processing_seconds) > 5
    for: 5m
    annotations:
      summary: "Agent taking > 5s"
  
  - alert: DatabaseConnectionPoolFull
    expr: moly_db_pool_available == 0
    annotations:
      summary: "Database connection pool exhausted"
```

---

## Logging

### Structured Logging

```go
// logging/logger.go
import "github.com/sirupsen/logrus"

logger = logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})

// Usage
logger.WithFields(logrus.Fields{
  "userId": userId,
  "conversationId": conversationId,
  "phase": "suggestions_generated",
  "processingTimeMs": 1247,
}).Info("Conversation processed")
```

### Log Aggregation

```bash
# Send logs to ELK Stack
# Filebeat → Logstash → Elasticsearch → Kibana

# Or send to Datadog
export DATADOG_LOGS_CONFIG=datadog.conf
```

### Log Levels

- **DEBUG**: Agent reasoning, decision trees
- **INFO**: API requests, completed tasks
- **WARN**: Slow operations (> 2s), deprecated features
- **ERROR**: Failed requests, agent errors
- **FATAL**: System-level failures

---

## Backup & Recovery

### Database Backups

```bash
# Full backup
pg_dump -h localhost -U postgres moly > moly_backup_$(date +%Y%m%d).sql

# Incremental backup with WAL archiving
# (Set up in PostgreSQL)

# Automated backups (daily)
0 2 * * * /usr/local/bin/backup-moly.sh

# Restore
psql -h localhost -U postgres moly < moly_backup_20260906.sql
```

### Disaster Recovery

- **RTO** (Recovery Time Objective): < 1 hour
- **RPO** (Recovery Point Objective): < 15 minutes

**Process**:
1. Identify failure (monitoring alert)
2. Restore from latest backup
3. Replay WAL logs (if < 15 min old)
4. Verify data integrity
5. Point extension to recovered backend

---

## Version Management

### Semantic Versioning

- **MAJOR** (1.0.0): Breaking API changes
- **MINOR** (1.1.0): New features (backward compatible)
- **PATCH** (1.1.1): Bug fixes

### Release Process

```bash
# Tag release
git tag -a v1.0.0 -m "Release 1.0.0"
git push origin v1.0.0

# Build docker image
docker build -t moly-backend:1.0.0 .
docker push moly-backend:1.0.0

# Deploy
kubectl set image deployment/moly-backend moly-backend=moly-backend:1.0.0 -n production
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/moly-backend -n production
kubectl rollout status deployment/moly-backend -n production
```

---

## Troubleshooting

### Backend Won't Start

```bash
# Check logs
journalctl -u moly-backend -f

# Verify database connection
psql $DATABASE_URL -c "SELECT 1"

# Check port
netstat -tulpn | grep 11436

# Verify environment variables
env | grep MOLY
```

### High Latency

```bash
# Check agent processing time
curl http://localhost:11436/api/status | jq .

# Monitor database queries
SELECT query, calls, total_time FROM pg_stat_statements ORDER BY total_time DESC;

# Check connection pool
SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;

# Profile with pprof
go tool pprof http://localhost:6060/debug/pprof/profile
```

### Memory Leaks

```bash
# Check memory usage
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Look for goroutine leaks
curl http://localhost:6060/debug/pprof/goroutine
```

### Database Connection Issues

```bash
# Increase pool size
DB_MAX_CONNECTIONS=50

# Use pgbouncer for connection pooling
# pgbouncer config:
[databases]
moly = host=localhost port=5432 dbname=moly

# Restart moly backend
systemctl restart moly-backend
```

---

## Maintenance

### Regular Tasks

- **Daily**: Check error rates, agent performance
- **Weekly**: Review logs for patterns, backup verification
- **Monthly**: Database maintenance (VACUUM, ANALYZE)
- **Quarterly**: Performance tuning, dependency updates

### Database Maintenance

```sql
-- Reclaim space
VACUUM FULL;

-- Update statistics
ANALYZE;

-- Remove old data
DELETE FROM messages WHERE created_at < NOW() - INTERVAL '1 year';

-- Rebuild indexes
REINDEX DATABASE moly;
```

### Dependency Updates

```bash
# Go dependencies
go get -u ./...
go mod tidy

# Security scanning
go list -json -m all | nancy sleuth

# Test thoroughly before deploying
go test ./...
```

---

## Disaster Scenarios

### Scenario 1: Database Down (Estimated Recovery: 10 min)

1. Check PostgreSQL status: `systemctl status postgresql`
2. Restart database: `systemctl restart postgresql`
3. Verify: `psql moly -c "SELECT 1"`
4. Restart backend: `systemctl restart moly-backend`
5. Monitor logs for recovery

### Scenario 2: API Service Stuck (Estimated Recovery: 5 min)

1. Check memory/CPU: `top`
2. Check connections: `curl http://localhost:6060/debug/pprof/goroutine`
3. Restart service: `systemctl restart moly-backend`
4. Check status: `curl http://localhost:11436/api/status`

### Scenario 3: Corrupted Data (Estimated Recovery: 1 hour)

1. Identify corruption with query anomalies
2. Restore from backup: `psql moly < backup.sql`
3. Replay WAL logs from backup point
4. Verify data integrity
5. Resume service

---

## Security Checklist

- [ ] API keys stored in environment, not in code
- [ ] Database password in encrypted secret manager
- [ ] HTTPS/TLS enabled in production
- [ ] CORS properly configured (no `*` origins)
- [ ] Input validation on all endpoints
- [ ] Rate limiting enabled
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS protection headers
- [ ] CSRF tokens if needed
- [ ] Regular security scanning
- [ ] Encrypted database backups
- [ ] Access logs with audit trail

---

## Capacity Planning

### Current Capacity

With 3 backend instances, 20 DB connections each:
- **API requests**: ~100 RPS
- **Concurrent users**: ~500
- **Database**: ~60 active connections

### Scaling Thresholds

When to scale:
- **CPU > 70%** → Add backend instance
- **DB connections > 80%** → Increase pool size
- **API latency > 1s p95** → Profile and optimize
- **Error rate > 1%** → Investigate and scale

---

This guide enables production-ready deployment and operations.
