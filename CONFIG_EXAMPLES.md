# Tenta Configuration Examples

## Basic Setup (Development)
```bash
# Minimal configuration - cache for 7 days with hourly pruning
tenta \
  --data-dir ./cache \
  --max-cache-age 168 \
  --cron-schedule "0 * * * *" \
  --http-port 8080 \
  --request-timeout 30 \
  --debug
```

## Production LAN Cache
```bash
# Production-grade LAN cache with metrics
tenta \
  --data-dir /mnt/cache/tenta \
  --max-cache-age 72 \
  --cron-schedule "0 2 * * *" \
  --http-port 8080 \
  --request-timeout 60 \
  --max-body-size 5368709120
```

## High-Performance Cache (Keep Everything)
```bash
# For scenarios where disk space is not an issue
tenta \
  --data-dir /mnt/large-cache \
  --max-cache-age 0 \
  --max-body-size 10737418240 \
  --request-timeout 120 \
  --http-port 8080
```

## Docker Deployment
```bash
docker run -d \
  --name tenta \
  --restart always \
  -v tenta-cache:/data \
  -p 8080:8080 \
  -p 2112:2112 \
  -e TENTA_MAX_CACHE_AGE=72 \
  -e TENTA_MAX_BODY_SIZE=5368709120 \
  -e TENTA_REQUEST_TIMEOUT=60 \
  ghcr.io/jeefy/tenta:main \
  --data-dir /data
```

## Using with systemd

### /etc/systemd/system/tenta.service
```ini
[Unit]
Description=Tenta LAN Cache
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=tenta
Group=tenta
WorkingDirectory=/var/lib/tenta
ExecStart=/usr/local/bin/tenta \
  --data-dir /var/cache/tenta \
  --max-cache-age 72 \
  --http-port 8080 \
  --request-timeout 60
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable tenta
sudo systemctl start tenta
```

## Monitoring with Prometheus

### prometheus.yml
```yaml
scrape_configs:
  - job_name: 'tenta'
    static_configs:
      - targets: ['localhost:2112']
    scrape_interval: 30s
```

### Useful Alerting Rules
```yaml
groups:
  - name: tenta.rules
    rules:
      - alert: TentaHighErrorRate
        expr: rate(tenta_errors[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High error rate in Tenta ({{ $value }})"

      - alert: TentaCacheFull
        expr: tenta_size > 1099511627776  # 1TB
        for: 10m
        annotations:
          summary: "Tenta cache is nearly full"

      - alert: TentaLowHitRate
        expr: tenta_hits / (tenta_hits + tenta_misses) < 0.5
        for: 30m
        annotations:
          summary: "Low cache hit rate in Tenta ({{ $value | humanizePercentage }})"
```

## Kubernetes Deployment with Persistent Storage

### pvc.yaml
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tenta-cache
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: standard
  resources:
    requests:
      storage: 100Gi
```

### deployment.yaml
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenta
  labels:
    app: tenta
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tenta
  template:
    metadata:
      labels:
        app: tenta
    spec:
      containers:
      - name: tenta
        image: ghcr.io/jeefy/tenta:main
        args:
          - --data-dir=/data
          - --max-cache-age=72
          - --http-port=8080
          - --request-timeout=60
          - --max-body-size=5368709120
          - --cron-schedule="0 2 * * *"
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 2112
          name: metrics
        volumeMounts:
        - mountPath: /data
          name: cache
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/cache/stats
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: cache
        persistentVolumeClaim:
          claimName: tenta-cache
```

### service.yaml
```yaml
apiVersion: v1
kind: Service
metadata:
  name: tenta
  labels:
    app: tenta
spec:
  selector:
    app: tenta
  type: LoadBalancer
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 2112
    targetPort: 2112
```

## Performance Tuning Tips

### Maximizing Cache Hit Rate
1. Set appropriate `--max-cache-age` based on content freshness requirements
2. Origin servers should emit proper Cache-Control headers
3. Use consistent hashing if running multiple instances
4. Monitor hit rate with: `tenta_hits / (tenta_hits + tenta_misses)`

### Handling Large Files
1. Increase `--max-body-size` as needed
2. Adjust `--request-timeout` for slow connections
3. Monitor disk space: `df -h /mnt/cache/tenta`
4. Check file count: `find /mnt/cache/tenta -type f | wc -l`

### Memory Optimization
- Tenta stores cache metadata in memory
- File count typically uses ~1KB per file
- Monitor with: `ps aux | grep tenta`

## Troubleshooting

### Check Cache Health
```bash
# Via API
curl http://localhost:8080/api/health | jq

# Check hit ratio
curl http://localhost:8080/api/cache/stats | jq '.hit_ratio'

# List large cached files
curl http://localhost:8080/api/cache/list | jq '.entries | sort_by(.size) | reverse | .[0:5]'
```

### Clear Problematic Cache
```bash
# Clear entire cache
curl -X DELETE http://localhost:8080/api/cache

# Delete specific file (if you know the hash)
curl -X DELETE http://localhost:8080/api/cache/delete/CACHE_KEY
```

### Debug Proxy Issues
```bash
# Check if proxy loop detection is working
curl -v -x http://localhost:8080 http://localhost:8080/api/health

# Should return 508 Loop Detected
```
