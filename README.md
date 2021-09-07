# Tenta

A lightweight high-performance LAN cache

## Features

* Built in scheduled pruning of files over a configured age
* Exported Prometheus metrics

```
> tenta --help
Fast and easy local LAN proxy cache

Usage:
  tenta [flags]

Flags:
      --cron-schedule string   Cron schedule to use for cleaning up cache files (default "* */1 * * *")
      --data-dir string        Directory to use for caching files (default "data/")
      --debug                  Enable debug logging
  -h, --help                   help for tenta
      --http-port int          Port to use for the HTTP server (default 8080)
      --max-cache-age int      Max age (in hours) of files. Value of 0 means no files will be deleted (default 0)
```

## Examples

### Docker

```sh
    docker run -d \
    -v path/to/storage:/data \
    -p 80:8080 -p 2112:2112
    jeefy/tenta:latest 
    --data-dir /data
    --debug
```

### Kubernetes

Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenta
  labels:
    app: tenta
spec:
  strategy:
    type: Recreate
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
        image: jeefy/tenta:latest
        args:
          - "--data-dir"
          - "/data"
        ports:
        - containerPort: 8080
        - containerPort: 2112
        volumeMounts:
        - mountPath: /data/
          name: lancache-data
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
      volumes:
      - name: lancache-data
        persistentVolumeClaim:
            claimName: lancache-data
```

## Additional Thoughts

**Will this handle HTTPS/SSL?**

This will not cache HTTPS requests. Much like monolithic, you will need to set up `sniproxy`.

**Why this over Monolithic?**

It started with: I wanted to export metrics on my cache to Prometheus and nginx wouldn't export raw Prometheus stats for their cache. Then it turned into "This would be a fun thing to write."

**Can you make it do X?**

Feature requests / optimizations / PRs are welcome! Feel free to ping me [@jeefy](https://twitter.com/jeefy) on Twitter.
