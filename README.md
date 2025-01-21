# rgw-exporter

## Prerequisites

Create an account with limited privileges:

```bash
radosgw-admin user create --uid="rgw-exporter" --display-name="RGW Usage Exporter"
radosgw-admin caps add --uid="rgw-exporter" --caps="metadata=read;usage=read;info=read;buckets=read;users=read"
```

## Install and Run

```sh
dpkg -i rgw-exporter_<version>_amd64.deb
```

Create config /etc/rgw-exporter/<realm>.yaml

```sh
systemctl start rgw-exportrr@<realm>.service
```

## Configuration file example

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
users_collector_enable: true
users_collector_interval: 3600
show_all_users: false
```

| Variable | Description | Default |
|----------|-------------|---------|
| access_key | "access key" value | |
| secret_key | "secret key" value | |
| endpoint | RGW endpoint url | |
| cluster_fsid | `ceph fsid` output | |
| cluster_name | Human-Readable name (upper case preferred) | |
| cluster_size | Cluster size in TB | |
| realm | realm name | |
| realm_vrf | Human-Readable realm name (upper case preferred) | |
| listen_ip | Bind IP | 127.0.0.1 |
| listen_port | Bind Port | 9240 |
| master_ip | collect stats if this IP is present on the server | 127.0.0.1 |
| usage_collector_interval | Ops statistics collection interval | 30 |
| buckets_collector_interval | Buckets statistics collection interval | 300 |
| rgw_connection_timeout | Connection timeout to RGW endpoint | 10 |
| insecure | Don't verify SSL certificate | false |
| users_collector_enable | Enable Users collector | false |
| users_collector_interval | Users Collector Interval | 3600 |
| show_all_users | Show all users info | false |

## Debug

Run the following command from the system terminal or shell:

```sh
rgw-exporter -c config.yaml
```

## Manual deployment recommendations

Create user

```sh
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

## Build recommendations

```sh
cd ..
git clone --branch fix_add_tenant https://github.com/krafZLorG/go-ceph.git

cd rgw-exporter
go mod init github.com/krafZLorG/rgw-exporter
go mod tidy
CGO_ENABLED=0 go build
```
