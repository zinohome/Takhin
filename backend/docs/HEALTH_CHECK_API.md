# Takhin Health Check API Documentation

This document describes the health check API endpoints available in Takhin Core and Console servers.

## Overview

Takhin provides three types of health check endpoints optimized for different use cases:

1. **Liveness Probe** (`/health/live`) - Checks if the application is alive
2. **Readiness Probe** (`/health/ready`) - Checks if the application is ready to accept traffic
3. **Detailed Health** (`/health`) - Comprehensive health status with component details

## Endpoints

### 1. Liveness Check

**Endpoint:** `GET /health/live`

**Purpose:** Determine if the application process is alive. Used by Kubernetes liveness probes.

**Response Codes:**
- `200 OK` - Application is alive

**Response Body:**
```json
{
  "alive": true
}
```

**Kubernetes Configuration:**
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 9091
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

**Use Case:** If this check fails, Kubernetes will restart the container.

---

### 2. Readiness Check

**Endpoint:** `GET /health/ready`

**Purpose:** Determine if the application is ready to accept traffic. Used by Kubernetes readiness probes.

**Response Codes:**
- `200 OK` - Application is ready
- `503 Service Unavailable` - Application is not ready (still initializing or dependencies unavailable)

**Response Body:**
```json
{
  "ready": true
}
```

**Readiness Criteria:**
- Topic manager initialized
- Coordinator initialized (Console only)
- Storage accessible

**Kubernetes Configuration:**
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 9091
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

**Use Case:** If this check fails, Kubernetes will remove the pod from service load balancers until it becomes ready.

---

### 3. Detailed Health Check

**Endpoint:** `GET /health`

**Purpose:** Get comprehensive health status including all components and system information.

**Response Codes:**
- `200 OK` - System is healthy or degraded but functional
- `503 Service Unavailable` - System is unhealthy

**Response Body:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h 15m 30s",
  "timestamp": "2026-01-06T09:00:00Z",
  "components": {
    "storage": {
      "status": "healthy",
      "message": "operating normally",
      "details": {
        "num_topics": 5,
        "num_partitions": 15,
        "total_size_mb": 1024.5
      }
    }
  },
  "system_info": {
    "go_version": "go1.21.0",
    "num_goroutines": 42,
    "num_cpu": 8,
    "memory_mb": 128.5
  }
}
```

**Status Values:**
- `healthy` - All components operating normally
- `degraded` - Some components not optimal but system functional
- `unhealthy` - Critical components failed

**Use Case:** Monitoring, debugging, and detailed status dashboards.

---

## Configuration

### Takhin Core (Kafka Server)

Health check server is configured in `configs/takhin.yaml`:

```yaml
health:
  enabled: true        # Enable health check HTTP server
  host: "0.0.0.0"     # Health check server host
  port: 9091          # Health check server port
```

**Environment Variables:**
```bash
TAKHIN_HEALTH_ENABLED=true
TAKHIN_HEALTH_HOST=0.0.0.0
TAKHIN_HEALTH_PORT=9091
```

### Takhin Console

Health check endpoints are built into the Console API server at `/api/health/*`:

```yaml
# No separate configuration needed - uses main API port
```

**Endpoints:**
- `GET /api/health/live`
- `GET /api/health/ready`
- `GET /api/health`

---

## Kubernetes Integration

### Complete Example

See `docs/examples/kubernetes/takhin-deployment.yaml` for full manifests.

**Key Points:**

1. **Separate Health Port:** Takhin Core uses port 9091 for health checks, separate from Kafka protocol (9092)

2. **Startup Probe:** Use for slow-starting containers:
```yaml
startupProbe:
  httpGet:
    path: /health/live
    port: 9091
  periodSeconds: 5
  failureThreshold: 30  # 150 seconds max startup time
```

3. **Liveness Probe:** Restart if unresponsive:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 9091
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
```

4. **Readiness Probe:** Control traffic routing:
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 9091
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
```

### Multi-Container Pods

If running multiple containers in a pod:

```yaml
containers:
- name: takhin
  ports:
  - name: kafka
    containerPort: 9092
  - name: health
    containerPort: 9091
  livenessProbe:
    httpGet:
      path: /health/live
      port: health  # Use named port
  readinessProbe:
    httpGet:
      path: /health/ready
      port: health
```

---

## Testing

### Local Testing

**Liveness Check:**
```bash
curl http://localhost:9091/health/live
# Response: {"alive":true}
```

**Readiness Check:**
```bash
curl http://localhost:9091/health/ready
# Response: {"ready":true}
```

**Detailed Health:**
```bash
curl http://localhost:9091/health | jq
# Response: Full health status JSON
```

### Kubernetes Testing

**Port Forward:**
```bash
kubectl port-forward deployment/takhin-broker 9091:9091
curl http://localhost:9091/health/live
```

**Check Probe Status:**
```bash
kubectl describe pod takhin-broker-<pod-id>
# Look for "Liveness" and "Readiness" probe results
```

**Watch Pod Events:**
```bash
kubectl get events --watch --field-selector involvedObject.name=takhin-broker-<pod-id>
```

---

## Monitoring Integration

### Prometheus

Health status can be monitored via metrics (port 9090):

```yaml
- job_name: 'takhin-health'
  static_configs:
  - targets: ['takhin-broker:9090']
  metrics_path: '/metrics'
```

### Custom Monitoring

Poll the detailed health endpoint:

```python
import requests
import time

while True:
    response = requests.get('http://takhin-broker:9091/health')
    health = response.json()
    
    if health['status'] != 'healthy':
        alert(f"Takhin unhealthy: {health}")
    
    time.sleep(30)
```

---

## Best Practices

1. **Use Startup Probes** for slow-starting applications to avoid premature restarts
2. **Set Appropriate Timeouts** - Don't make them too short or too long
3. **Adjust Failure Thresholds** based on your recovery time objectives
4. **Monitor Probe Failures** in production to detect issues early
5. **Test Probes Locally** before deploying to Kubernetes
6. **Use Named Ports** in pod specs for better readability
7. **Separate Health Port** from application traffic for isolation

---

## Troubleshooting

### Pod Keeps Restarting

**Check liveness probe:**
```bash
kubectl logs takhin-broker-<pod-id> --previous
kubectl describe pod takhin-broker-<pod-id>
```

**Common causes:**
- `initialDelaySeconds` too short
- Application startup slower than expected
- Health endpoint not responding
- Port misconfiguration

**Solution:** Increase `initialDelaySeconds` or add startup probe.

### Pod Not Receiving Traffic

**Check readiness probe:**
```bash
kubectl get pod takhin-broker-<pod-id> -o wide
# Look for READY column (should be 1/1)

kubectl describe pod takhin-broker-<pod-id>
# Check "Conditions" section for Ready status
```

**Common causes:**
- Dependencies not initialized
- Storage not accessible
- Port misconfiguration

**Solution:** Check application logs and ensure all dependencies are ready.

### Health Endpoint Returns 503

**Check component status:**
```bash
curl http://localhost:9091/health | jq '.components'
```

Look for components with `unhealthy` status and check their messages.

---

## API Reference

### Response Schemas

**HealthCheck (Detailed):**
```typescript
{
  status: "healthy" | "degraded" | "unhealthy",
  version: string,
  uptime: string,
  timestamp: string (ISO 8601),
  components: {
    [key: string]: {
      status: "healthy" | "degraded" | "unhealthy",
      message?: string,
      details?: { [key: string]: any }
    }
  },
  system_info: {
    go_version: string,
    num_goroutines: number,
    num_cpu: number,
    memory_mb: number
  }
}
```

**LivenessCheck:**
```typescript
{
  alive: boolean
}
```

**ReadinessCheck:**
```typescript
{
  ready: boolean
}
```

---

## Additional Resources

- [Kubernetes Probes Documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Takhin Metrics Documentation](../monitoring/metrics.md)
- [Deployment Examples](../examples/kubernetes/)
