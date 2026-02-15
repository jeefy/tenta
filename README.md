# Tenta

A lightweight high-performance LAN cache proxy with Prometheus metrics and REST API management.

## Features

* **HTTP Caching Proxy** - Fast LAN-based caching for HTTP requests with automatic origin fetching
* **Prometheus Metrics** - Built-in metrics export on port 2112
* **Scheduled Pruning** - Automatic cleanup of cached files older than specified duration
* **REST API** - Full-featured API for cache management and monitoring
* **Health Checks** - Service health endpoint for monitoring
* **Cache Control Aware** - Respects Cache-Control headers to determine cacheability
* **Response Type Tracking** - Separate metrics for cache hits, misses, 404s, and errors
* **Request Timeouts** - Configurable timeouts for upstream requests
* **Size Limits** - Configurable maximum size for cached responses
* **Steam Support** - Special handling for Steam CDN requests

## Quick Start

### Docker

```sh
docker run -d \
  -v /path/to/cache:/data \
  -p 8080:8080 \
  -p 2112:2112 \
  ghcr.io/jeefy/tenta:main \
  --data-dir /data \
  --max-cache-age 72 \
  --http-port 8080
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenta
spec:
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
          - "--data-dir=/data"
          - "--max-cache-age=72"
          - "--http-port=8080"
          - "--request-timeout=30"
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 2112
          name: metrics
        volumeMounts:
        - mountPath: /data
          name: cache-data
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: cache-data
        persistentVolumeClaim:
          claimName: tenta-cache
```

## Configuration

### Command-Line Flags

```
Flags:
  --cron-schedule string      Cron schedule for cache cleanup (default "* */1 * * *")
  --data-dir string           Directory for cached files (default "data/")
  --debug                     Enable debug logging
  --http-port int             HTTP server port (default 8080)
  --max-cache-age int         Max cache file age in hours, 0=unlimited (default 0)
  --max-body-size int         Max response size to cache in bytes (default 1073741824)
  --request-timeout int       Timeout for upstream requests in seconds (default 30)
```

### Environment Variables

Configure via environment variables by prefixing flag names with `TENTA_` and converting to UPPER_CASE:
- `TENTA_DATA_DIR=/var/cache/tenta`
- `TENTA_MAX_CACHE_AGE=72`
- `TENTA_HTTP_PORT=8080`

### Configuration Examples

**Basic LAN Cache (72-hour retention)**
```bash
tenta \
  --data-dir /var/cache/tenta \
  --max-cache-age 72 \
  --cron-schedule "0 2 * * *"
```

**High-Performance Mode (keep everything, hourly checks)**
```bash
tenta \
  --data-dir /mnt/large-cache \
  --max-cache-age 0 \
  --max-body-size 5368709120 \
  --request-timeout 60 \
  --http-port 8080
```

**Aggressive Caching (clean daily, smaller files)**
```bash
tenta \
  --data-dir /var/cache/tenta \
  --max-cache-age 24 \
  --max-body-size 536870912 \
  --cron-schedule "0 3 * * *"
```

## REST API

### Health Check

**GET /api/health** - Service status and configuration

Response:
```json
{
  "status": "healthy",
  "uptime": "2h30m15s",
  "data_dir": "/data",
  "cache_files": 1234,
  "cache_size_bytes": 5368709120,
  "max_cache_age_hours": 72,
  "cron_schedule": "* */1 * * *"
}
```

### Cache Statistics

**GET /api/cache/stats** - Current cache performance metrics

Response:
```json
{
  "total_requests": 50000,
  "cache_hits": 45000,
  "cache_misses": 5000,
  "hit_ratio": 0.9,
  "file_count": 1234,
  "cache_size_bytes": 5368709120
}
```

### List Cached Files

**GET /api/cache/list** - List all cached files

Response:
```json
{
  "count": 2,
  "entries": [
    {
      "filename": "1234567890123456789",
      "size": 1048576,
      "mod_time": "2024-02-14 22:00:00 +0000 UTC"
    }
  ]
}
```

### Cache Info

**GET /api/cache/info** - Cache size distribution

Response:
```json
{
  "total_files": 1234,
  "size_distribution": {
    "small_<1mb": 800,
    "medium_1-10mb": 300,
    "large_10-100mb": 100,
    "huge_>100mb": 34
  }
}
```

### Clear Cache

**DELETE /api/cache** - Remove all cached files

Response:
```json
{
  "status": "cleared",
  "deleted": 1234,
  "size_freed": 5368709120
}
```

**DELETE /api/cache/delete/{cache_key}** - Remove specific cached file

Response:
```json
{
  "status": "deleted",
  "key": "1234567890123456789",
  "size": 1048576
}
```

## Prometheus Metrics

Metrics are exported on port 2112 at `/metrics`. Key metrics:

- `tenta_requests_received` - Total HTTP requests handled
- `tenta_hits` - Cache hits
- `tenta_misses` - Cache misses
- `tenta_files` - Number of files in cache
- `tenta_size` - Total cache size in bytes
- `tenta_errors` - Total errors
- `tenta_not_found` - 404 responses
- `tenta_server_errors` - 5xx responses

### Example Queries

```promql
# Cache hit ratio
tenta_hits / tenta_requests_received

# Average file size
tenta_size / tenta_files

# Requests per second
rate(tenta_requests_received[1m])
```

## Usage Examples

### Using as HTTP Proxy

Configure your client to use Tenta as HTTP proxy:
```bash
export http_proxy=http://tenta-host:8080
export https_proxy=http://tenta-host:8080  # HTTPS proxying requires sniproxy
curl http://example.com/large-file.iso
```

### Monitoring with Docker Compose

```yaml
version: '3'
services:
  tenta:
    image: ghcr.io/jeefy/tenta:main
    ports:
      - "8080:8080"
      - "2112:2112"
    volumes:
      - cache-data:/data
    environment:
      - TENTA_MAX_CACHE_AGE=72
      - TENTA_MAX_BODY_SIZE=5368709120
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'

volumes:
  cache-data:
```

## Performance Tuning

### Cache Hit Ratio

Tenta respects HTTP Cache-Control headers. To maximize cache hits:
- Configure appropriate TTLs on your origin servers
- Use immutable assets (hash-based file names) when possible
- Monitor cache hit ratio with Prometheus queries

### Large Files

For large file serving (> 1GB):
- Increase `--max-body-size`
- Ensure sufficient disk space
- Monitor disk I/O

### Multiple Instances

For high-traffic scenarios, run multiple Tenta instances behind a load balancer:
- Each instance maintains its own cache
- Use consistent hashing to maximize cache locality
- Monitor metrics from each instance

## Troubleshooting

### Proxy Loops
If you see "Proxy loop detected" errors, ensure:
- DNS resolution doesn't point to Tenta for origin servers
- The `tenta-proxy` header is being honored
- Your DNS configuration is correct

### Low Cache Hit Rate
Check:
- Cache-Control headers on origin responses
- Maximum cache size limits
- Cron schedule (files might be pruned too aggressively)

### High Memory Usage
- Check file count and size with `/api/cache/info`
- Reduce `--max-body-size` if storing too many large files
- Increase `--max-cache-age` to keep cache fresh

## HTTPS/SSL

Tenta cannot directly cache HTTPS requests due to encryption. To cache HTTPS content:
1. Set up `sniproxy` as a transparent HTTPS proxy
2. Route HTTPS traffic through sniproxy to Tenta
3. sniproxy handles the TLS termination

## License

See LICENSE file for details.

## Contributing

Feature requests and PRs are welcome! Please open an issue first to discuss changes.

## Support

For questions and issues, please open a GitHub issue or contact [@jeefy](https://twitter.com/jeefy) on Twitter.
