# slowpg

## Simple PG slow query exporter

> Build it and push into your docker registry

![Alt text](./images/db.png?raw=true "Screen1")


> Prometheus scrap config 

```yaml
      - job_name: "slowpg-dev"
        static_configs:
          - targets: ["slowpg.develop:8080"]
            labels:
              instance: "slowpg-dev"
```