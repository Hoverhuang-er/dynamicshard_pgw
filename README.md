# Dynamic Shard for Prometheus PushGateway HA with Service discovery


## Architecture graph

### Relabeling with PushGateway

- For Non-grouping metrics uri
```
http://pushgateway_addr/metrics/job/<JOB_NAME>
```
- For grouping metrics uri
```
http://pushgateway_addr/metrics/job/<JOB_NAME>/<LABEL_NAME>/<LABEL_VALUE>
```
- Which is different with POST and PUT method, PUT only replace metrics with same job, POST replace metrics with same job and label
## Why need dynamic shard

### If simple pgw HA with lb
- use rr lb: if push data to multi pgw instances, prometheus scrape will cause data error, like this
- root cause is at t1 time, the value of metric is 10, at t2 time, the value is 20, but prometheus scrape the data at t1 and t2, the value is 10 and 10, which is not correct, the value should be 10 and 20
### Only use static hash and prometheus static config pgw
- Such as 3 replicas pgw, lb use consistent hash to hash request_uri
- promethues scrape for 3 node of pushgateway with static config
```
  - job_name: pushgateway
    honor_labels: true
    honor_timestamps: true
    scrape_interval: 5s
    scrape_timeout: 4s
    metrics_path: /metrics
    scheme: http
    static_configs:
    - targets:
      - pgw-a:9091
      - pgw-b:9091

```
- Finaly result is that we can do hash sharding, but we can not solve the problem that when one pgw instance is down, the request hash to this instance will fail

#### Solution A: dynamic hash sharding + consul service_check
- Dyshard would start and register pgw service to consul
- Consul check service will check Dyshard server health with scheduler
- Each push request will be consistence hashed by request path, like this
```
##### Job different
- http://pushgateway_addr/metrics/job/job_a
- http://pushgateway_addr/metrics/job/job_b
- http://pushgateway_addr/metrics/job/job_c
##### Label different
- http://pushgateway_addr/metrics/job/job_a/tag_a/value_a
- http://pushgateway_addr/metrics/job/job_a/tag_a/value_b
```
- When Pushgateway instances are oom or abnormal restart, consul check service will mark the bad instance as down
~~Dyshard would watch pgw node count change~~
- rebuild consistent hash ring, rehash job
- Prometheus use consul service discovery to get Pushgateway instances, no need to change prometheus config
- When it use redirect, it will not handle the request, which is simple and efficient
- Dyshard is stateless, it can start multiple instances as the entrance of traffic and Pushgateway instances
- When scaling up, all existing Pushgateway instances need to be restarted at the same time
- The shortcomings are that it does not solve the problem of promethues single point and sharding, but prometheus single point can slove with [VictoriaMetrics](https://victoriametrics.com/)

### How to use
   
#### Compile and run
```shell script
$ git clone https://github.com/Hoverhuang-er/dynamicshard_pgw
$ cd  dynamicshard_pgw && make
```

#### Start Dyshard service

```shell
./dyshard
```
#### Injection dyshard with prometheus
```yaml
scrape_configs:
  - job_name: pushgateway
    consul_sd_configs:
      - server: $cousul_api
        services:
          - pushgateway
    relabel_configs:
    - source_labels:  ["__meta_consul_dc"]
      target_label: "dc"

```