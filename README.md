# rgw-exporter

## Prerequisites

Create an account with limited privileges:

```bash
radosgw-admin user create --uid="rgw-exporter" --display-name="RGW Usage Exporter"
radosgw-admin caps add --uid="rgw-exporter" --caps="metadata=read;usage=read;info=read;buckets=read;users=read"
```

## Installation

```sh
dpkg -i rgw-exporter_<version>_amd64.deb
```

## Configuration

Create a configuration file at: 
```
/etc/rgw-exporter/<realm>.yaml
```

### Example configuration with default values:

```yaml
access_key: "access"
secret_key: "secret"
endpoint: http://127.0.0.1:8080
cluster_fsid: 00000000-0000-0000-0000-000000000000
cluster_name: DEFAULT
cluster_size: 1
realm: default
realm_vrf: DEFAULT
listen_ip: 127.0.0.1
listen_port: 9240
master_ip: 127.0.0.1
rgw_connection_timeout: 60
rgw_connection_check_ssl: false
usage_skip_without_bucket: false
usage_collector_interval: 30
buckets_collector_interval: 300
lc_collector_enable: false
lc_collector_interval: 28800
```

## Running

Run the rgw-exporter manually:

```sh
rgw-exporter -c config.yaml
```

### Debug mode

```sh
rgw-exporter -d -c config.yaml
```

## Systemd service

Start and enable the rgw-exporter as a service:

```sh
systemctl start rgw-exportrr@<realm>.service
systemctl enable rgw-exportrr@<realm>.service
```

Example `rgw-exporter@.service`:

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

## Manual deployment recommendations

Create the rgw-exporter system user:

```sh
useradd -r -M -d /nonexistent -s /usr/sbin/nologin rgw-exporter
```

## Build Instructions

```shell
cd rgw-exporter
go mod init github.com/krafZLorG/rgw-exporter
go mod tidy
CGO_ENABLED=0 go build
```

## Build and Package as deb

```shell
cd deb
./build-package <version>

ls -l deb
```