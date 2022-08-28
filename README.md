# CephFS exporter

[![build](https://github.com/dankeder/prometheus-cephfs-exporter/actions/workflows/build.yaml/badge.svg)](https://github.com/dankeder/prometheus-cephfs-exporter/actions/workflows/build.yaml)

Prometheus exporter for CephFS metrics. Currently exposed metrics:

- `cephfs_subvolume_quota_bytes` for all subvolumes of all CephFS volumes
- `cephfs_subvolume_used_bytes` for all subvolumes of all CephFS volumes


## Build

```
make build
```


## Usage

Run, optionally specify address/port where to listen for requests:

```
./cephfs_exporter -web.listen-address :9939
```

`cephfs_exporter` is using the default ceph configuration file in
`/etc/ceph/ceph.conf`. If ceph management commands work correctly the exporter
should also work.
