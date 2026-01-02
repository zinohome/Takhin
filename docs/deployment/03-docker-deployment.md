# Takhin Docker & Kubernetes Deployment Guide

This guide covers deploying Takhin using Docker containers and Kubernetes orchestration.

## Docker Deployment

### Building Docker Image

#### Create Dockerfile

Create `Dockerfile` in project root:

```dockerfile
# Multi-stage build for smaller image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /takhin ./cmd/takhin
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /takhin-console ./cmd/console

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /takhin /usr/local/bin/takhin
COPY --from=builder /takhin-console /usr/local/bin/takhin-console

# Copy default configuration
COPY backend/configs/takhin.yaml /etc/takhin/takhin.yaml

# Create data directory
RUN mkdir -p /var/lib/takhin/data

# Expose ports
EXPOSE 9092 9090 8080

# Run Takhin by default
CMD ["/usr/local/bin/takhin", "-config", "/etc/takhin/takhin.yaml"]
```

#### Build Image

```bash
# Build image
docker build -t takhin:latest -f Dockerfile .

# Build with specific version tag
docker build -t takhin:1.0.0 -f Dockerfile .

# Build using Task
task docker:build
```

### Running Single Container

```bash
# Run Takhin server
docker run -d \
  --name takhin \
  -p 9092:9092 \
  -p 9090:9090 \
  -v takhin-data:/var/lib/takhin/data \
  -e TAKHIN_LOGGING_LEVEL=info \
  takhin:latest

# Check logs
docker logs -f takhin

# Stop container
docker stop takhin
```

### Docker Compose Deployment

#### Standalone Mode

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  takhin:
    image: takhin:latest
    container_name: takhin-server
    ports:
      - "9092:9092"
      - "9090:9090"
    volumes:
      - takhin-data:/var/lib/takhin/data
      - ./configs/takhin.yaml:/etc/takhin/takhin.yaml:ro
    environment:
      - TAKHIN_LOGGING_LEVEL=info
      - TAKHIN_KAFKA_ADVERTISED_HOST=localhost
    restart: unless-stopped
    networks:
      - takhin-net

  console:
    image: takhin:latest
    container_name: takhin-console
    command: ["/usr/local/bin/takhin-console", "-data-dir", "/var/lib/takhin/data", "-api-addr", ":8080"]
    ports:
      - "8080:8080"
    volumes:
      - takhin-data:/var/lib/takhin/data:ro
    environment:
      - ENABLE_AUTH=true
      - API_KEYS=your-secret-key-here
    depends_on:
      - takhin
    restart: unless-stopped
    networks:
      - takhin-net

volumes:
  takhin-data:

networks:
  takhin-net:
    driver: bridge
```

Start services:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Remove volumes (data will be lost)
docker-compose down -v
```

#### Cluster Mode (3 Nodes)

Create `docker-compose.cluster.yml`:

```yaml
version: '3.8'

services:
  broker1:
    image: takhin:latest
    container_name: takhin-broker1
    hostname: broker1
    ports:
      - "9092:9092"
      - "9090:9090"
    volumes:
      - broker1-data:/var/lib/takhin/data
      - ./configs/broker1.yaml:/etc/takhin/takhin.yaml:ro
    environment:
      - TAKHIN_KAFKA_BROKER_ID=1
      - TAKHIN_KAFKA_CLUSTER_BROKERS=[1,2,3]
      - TAKHIN_KAFKA_ADVERTISED_HOST=broker1
      - TAKHIN_REPLICATION_DEFAULT_REPLICATION_FACTOR=3
    networks:
      - takhin-cluster
    restart: unless-stopped

  broker2:
    image: takhin:latest
    container_name: takhin-broker2
    hostname: broker2
    ports:
      - "9093:9092"
      - "9091:9090"
    volumes:
      - broker2-data:/var/lib/takhin/data
      - ./configs/broker2.yaml:/etc/takhin/takhin.yaml:ro
    environment:
      - TAKHIN_KAFKA_BROKER_ID=2
      - TAKHIN_KAFKA_CLUSTER_BROKERS=[1,2,3]
      - TAKHIN_KAFKA_ADVERTISED_HOST=broker2
      - TAKHIN_REPLICATION_DEFAULT_REPLICATION_FACTOR=3
    networks:
      - takhin-cluster
    restart: unless-stopped

  broker3:
    image: takhin:latest
    container_name: takhin-broker3
    hostname: broker3
    ports:
      - "9094:9092"
      - "9092:9090"
    volumes:
      - broker3-data:/var/lib/takhin/data
      - ./configs/broker3.yaml:/etc/takhin/takhin.yaml:ro
    environment:
      - TAKHIN_KAFKA_BROKER_ID=3
      - TAKHIN_KAFKA_CLUSTER_BROKERS=[1,2,3]
      - TAKHIN_KAFKA_ADVERTISED_HOST=broker3
      - TAKHIN_REPLICATION_DEFAULT_REPLICATION_FACTOR=3
    networks:
      - takhin-cluster
    restart: unless-stopped

volumes:
  broker1-data:
  broker2-data:
  broker3-data:

networks:
  takhin-cluster:
    driver: bridge
```

Start cluster:

```bash
docker-compose -f docker-compose.cluster.yml up -d
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster 1.24+ (minikube, EKS, GKE, AKS)
- kubectl configured
- Helm 3.x (optional, for Helm chart deployment)

### Namespace Setup

```bash
# Create namespace
kubectl create namespace takhin

# Set as default
kubectl config set-context --current --namespace=takhin
```

### ConfigMap for Configuration

Create `k8s/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: takhin-config
  namespace: takhin
data:
  takhin.yaml: |
    server:
      host: "0.0.0.0"
      port: 9092
    
    kafka:
      broker:
        id: 1
      cluster:
        brokers: [1, 2, 3]
      listeners:
        - "tcp://0.0.0.0:9092"
      advertised:
        host: "takhin-0.takhin-headless.takhin.svc.cluster.local"
        port: 9092
      max:
        message:
          bytes: 10485760
        connections: 5000
    
    storage:
      data:
        dir: "/var/lib/takhin/data"
      log:
        segment:
          size: 2147483648
        retention:
          hours: 168
    
    replication:
      default:
        replication:
          factor: 3
    
    logging:
      level: "info"
      format: "json"
    
    metrics:
      enabled: true
      host: "0.0.0.0"
      port: 9090
      path: "/metrics"
```

Apply ConfigMap:

```bash
kubectl apply -f k8s/configmap.yaml
```

### StatefulSet Deployment

Create `k8s/statefulset.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: takhin-headless
  namespace: takhin
  labels:
    app: takhin
spec:
  clusterIP: None
  ports:
    - port: 9092
      name: kafka
    - port: 9090
      name: metrics
    - port: 7946
      name: raft
  selector:
    app: takhin
---
apiVersion: v1
kind: Service
metadata:
  name: takhin-service
  namespace: takhin
  labels:
    app: takhin
spec:
  type: LoadBalancer
  ports:
    - port: 9092
      targetPort: 9092
      name: kafka
    - port: 9090
      targetPort: 9090
      name: metrics
  selector:
    app: takhin
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: takhin
  namespace: takhin
spec:
  serviceName: takhin-headless
  replicas: 3
  selector:
    matchLabels:
      app: takhin
  template:
    metadata:
      labels:
        app: takhin
    spec:
      containers:
      - name: takhin
        image: takhin:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 9092
          name: kafka
        - containerPort: 9090
          name: metrics
        - containerPort: 7946
          name: raft
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: TAKHIN_KAFKA_BROKER_ID
          value: "$(echo ${POD_NAME} | awk -F'-' '{print $NF + 1}')"
        - name: TAKHIN_KAFKA_ADVERTISED_HOST
          value: "$(POD_NAME).takhin-headless.$(POD_NAMESPACE).svc.cluster.local"
        volumeMounts:
        - name: data
          mountPath: /var/lib/takhin/data
        - name: config
          mountPath: /etc/takhin
        livenessProbe:
          httpGet:
            path: /metrics
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          tcpSocket:
            port: 9092
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
      volumes:
      - name: config
        configMap:
          name: takhin-config
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "standard"  # Change to your StorageClass
      resources:
        requests:
          storage: 100Gi
```

Deploy StatefulSet:

```bash
kubectl apply -f k8s/statefulset.yaml

# Check status
kubectl get pods -w

# Check logs
kubectl logs takhin-0 -f
```

### Console Deployment

Create `k8s/console-deployment.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: takhin-console
  namespace: takhin
spec:
  type: LoadBalancer
  ports:
    - port: 8080
      targetPort: 8080
      name: http
  selector:
    app: takhin-console
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: takhin-console
  namespace: takhin
spec:
  replicas: 2
  selector:
    matchLabels:
      app: takhin-console
  template:
    metadata:
      labels:
        app: takhin-console
    spec:
      containers:
      - name: console
        image: takhin:latest
        command: ["/usr/local/bin/takhin-console"]
        args:
          - "-data-dir=/var/lib/takhin/data"
          - "-api-addr=:8080"
          - "-enable-auth=true"
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: API_KEYS
          valueFrom:
            secretKeyRef:
              name: takhin-secrets
              key: api-keys
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

Create Secret for API keys:

```bash
kubectl create secret generic takhin-secrets \
  --from-literal=api-keys=your-secret-key-here \
  -n takhin
```

Deploy Console:

```bash
kubectl apply -f k8s/console-deployment.yaml
```

### Ingress Configuration (Optional)

Create `k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: takhin-ingress
  namespace: takhin
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: takhin.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: takhin-console
            port:
              number: 8080
  - host: kafka.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: takhin-service
            port:
              number: 9092
```

Apply Ingress:

```bash
kubectl apply -f k8s/ingress.yaml
```

## Helm Chart Deployment

### Create Helm Chart Structure

```bash
mkdir -p takhin-chart/templates
cd takhin-chart
```

Create `Chart.yaml`:

```yaml
apiVersion: v2
name: takhin
description: A Helm chart for Takhin Kafka-compatible streaming platform
type: application
version: 1.0.0
appVersion: "1.0.0"
```

Create `values.yaml`:

```yaml
replicaCount: 3

image:
  repository: takhin
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: LoadBalancer
  kafka:
    port: 9092
  metrics:
    port: 9090
  console:
    port: 8080

persistence:
  enabled: true
  storageClass: "standard"
  size: 100Gi

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 1000m
    memory: 2Gi

config:
  logging:
    level: info
  replication:
    factor: 3
  storage:
    segmentSize: 2147483648
    retentionHours: 168

console:
  enabled: true
  replicas: 2
  auth:
    enabled: true
    apiKeys: "changeme"
```

### Install with Helm

```bash
# Install chart
helm install takhin ./takhin-chart -n takhin

# Upgrade
helm upgrade takhin ./takhin-chart -n takhin

# Uninstall
helm uninstall takhin -n takhin
```

## Monitoring in Kubernetes

### Prometheus ServiceMonitor

Create `k8s/servicemonitor.yaml`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: takhin
  namespace: takhin
spec:
  selector:
    matchLabels:
      app: takhin
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

## Scaling Operations

### Horizontal Scaling

```bash
# Scale StatefulSet
kubectl scale statefulset takhin --replicas=5 -n takhin

# Scale Console
kubectl scale deployment takhin-console --replicas=3 -n takhin
```

### Vertical Scaling (Resource Limits)

```bash
# Update resource limits
kubectl patch statefulset takhin -n takhin -p '{"spec":{"template":{"spec":{"containers":[{"name":"takhin","resources":{"limits":{"memory":"8Gi","cpu":"4000m"}}}]}}}}'
```

## Backup and Recovery

### Volume Snapshots

```bash
# Create VolumeSnapshot (requires VolumeSnapshot CRD)
kubectl apply -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: takhin-snapshot-$(date +%Y%m%d)
  namespace: takhin
spec:
  volumeSnapshotClassName: csi-snapclass
  source:
    persistentVolumeClaimName: data-takhin-0
EOF
```

### Restore from Snapshot

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-takhin-0-restored
spec:
  dataSource:
    name: takhin-snapshot-20260102
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
```

## Troubleshooting

### Common Issues

```bash
# Check pod status
kubectl get pods -n takhin

# View logs
kubectl logs takhin-0 -n takhin -f

# Describe pod for events
kubectl describe pod takhin-0 -n takhin

# Check PVC status
kubectl get pvc -n takhin

# Execute commands in pod
kubectl exec -it takhin-0 -n takhin -- /bin/sh

# Port forward for local testing
kubectl port-forward svc/takhin-service 9092:9092 -n takhin
```

## Next Steps

- [Configuration Reference](./04-configuration-reference.md)
- [Troubleshooting Guide](./05-troubleshooting.md)
- [Monitoring Best Practices](../operations/monitoring.md)
