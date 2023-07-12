# rgw-exporter

## Prerequisites

Create an account with limited privileges:

```bash
radosgw-admin user create --uid="rgw-exporter" --display-name="RGW Usage Exporter"
radosgw-admin caps add --uid="rgw-exporter" --caps="metadata=read;usage=read;info=read;buckets=read;users=read"
```

Configuration file example:

```yaml
access_key: 
secret_key: 
endpoint: 
cluster_fsid:
cluster_name:
cluster_size:
realm: 
realm_vrf:
listen_ip:
listen_port: 
master_ip:
usage_collector_interval: 
buckets_collector_interval:
rgw_connection_timeout: 
insecure: false
```

| Variable | Description | Default |
|----------|-------------|---------|
| access_key | rgw admin ops access key | |
| secret_key | rgw admin ops secret key | |
| access_key | | |
| secret_key | | |
| endpoint | | |
| cluster_fsid | | |
| cluster_name | | |
| cluster_size | | |
| realm | | |
| realm_vrf | | |
| listen_ip | | 127.0.0.1 |
| listen_port | | 9240 |
| master_ip | | 127.0.0.1 |
| usage_collector_interval | | 30 |
| buckets_collector_interval | | 300 |
| rgw_connection_timeout | | 10 |
| insecure | | false |

## Launch

Run the following command from the system terminal or shell:

```bash
rgw-exporter -c config.yaml
```

## Deployment Recommendations

Create user

```bash
useradd -r -M -d /nonexistent -s /usr/sbin/nologin rgw-exporter
```

`rgw-exporter@.service` example

```systemd.unit
[Unit]
Description=RGW Usage Exporter
After=network.target
ConditionPathExists=/etc/rgw-exporter/%i.yaml
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStartPre=/bin/bash -c '/bin/sleep $((RANDOM % 15))'
ExecStart=/usr/local/bin/rgw-exporter -c /etc/rgw-exporter/%i.yaml
Restart=on-failure
RestartSec=5s
User=rgw-exporter
Group=rgw-exporter


[Install]
WantedBy=multi-user.target
```
