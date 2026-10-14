package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/ceph/go-ceph/cephfs/admin"
	"github.com/ceph/go-ceph/rados"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "cephfs"

var (
	listenAddress = flag.String("web.listen-address", ":9939", "Address to listen on")

	printVersion = flag.Bool("version", false, "Print version")

	labels      = []string{"volume", "group", "subvolume"}
	groupLabels = []string{"volume", "group"}

	cephfsSubvolumeUsedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "subvolume", "used_bytes"),
		"Used space in a CephFS subvolume.",
		labels,
		nil,
	)

	cephfsSubvolumeQuotaBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "subvolume", "quota_bytes"),
		"Size (quota) of the CephFS subvolume.",
		labels,
		nil,
	)

	cephfsSubvolumeGroupUsedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "subvolumegroup", "used_bytes"),
		"Used space in a CephFS subvolume group.",
		groupLabels,
		nil,
	)

	cephfsSubvolumeGroupQuotaBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "subvolumegroup", "quota_bytes"),
		"Size (quota) of the CephFS subvolume group.",
		groupLabels,
		nil,
	)

	version = "unknown"
)

type CephFSExporter struct{}

func NewCephFSExporter() *CephFSExporter {
	return &CephFSExporter{}
}

func (e *CephFSExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- cephfsSubvolumeUsedBytes
	ch <- cephfsSubvolumeQuotaBytes
	ch <- cephfsSubvolumeGroupUsedBytes
	ch <- cephfsSubvolumeGroupQuotaBytes
}

func newFSAdmin() (*admin.FSAdmin, error) {
	conn, err := rados.NewConn()
	if err != nil {
		return nil, err
	}
	if err := conn.ReadDefaultConfigFile(); err != nil {
		return nil, err
	}
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	return admin.NewFromConn(conn), nil
}

func (e *CephFSExporter) Collect(ch chan<- prometheus.Metric) {
	fsa, err := newFSAdmin()
	if err != nil {
		log.Fatal(err)
	}

	// Get CephFS volumes
	volumes, err := fsa.ListVolumes()
	if err != nil {
		log.Fatalf("cephfs volume list failed: %s", err)
	}
	log.Printf("DEBUG: found %d volume(s): %v", len(volumes), volumes)

	for _, volume := range volumes {
		// Get CephFS subvolume groups (named groups only, default group "" is added separately)
		groups, err := fsa.ListSubVolumeGroups(volume)
		if err != nil {
			log.Fatalf("cephfs subvolume group list failed: %s", err)
		}
		log.Printf("DEBUG: volume %q: found %d group(s): %v", volume, len(groups), groups)

		// Always include the default group
		groups = append([]string{""}, groups...)

		for _, group := range groups {
			// Get CephFS subvolume group info (only for named groups)
			if group != "" {
				groupInfo, err := fsa.SubVolumeGroupInfo(volume, group)
				if err != nil {
					log.Fatalf("cephfs subvolumegroup info failed: %s", err)
				}
				log.Printf("DEBUG: volume %q group %q: used=%d", volume, group, groupInfo.BytesUsed)

				ch <- prometheus.MustNewConstMetric(cephfsSubvolumeGroupUsedBytes, prometheus.GaugeValue, float64(groupInfo.BytesUsed), volume, group)

				groupBytesQuota, ok := groupInfo.BytesQuota.(admin.ByteCount)
				if ok {
					ch <- prometheus.MustNewConstMetric(cephfsSubvolumeGroupQuotaBytes, prometheus.GaugeValue, float64(groupBytesQuota), volume, group)
				}
			}

			// Get CephFS subvolumes
			subvolumes, err := fsa.ListSubVolumes(volume, group)
			if err != nil {
				log.Fatalf("cephfs subvolume list failed: %s", err)
			}
			log.Printf("DEBUG: volume %q group %q: found %d subvolume(s): %v", volume, group, len(subvolumes), subvolumes)

			for _, subvolume := range subvolumes {
				subvolumeInfo, err := fsa.SubVolumeInfo(volume, group, subvolume)
				if err != nil {
					log.Fatalf("cephfs subvolume info failed: %s", err)
				}

				ch <- prometheus.MustNewConstMetric(cephfsSubvolumeUsedBytes, prometheus.GaugeValue, float64(subvolumeInfo.BytesUsed), volume, group, subvolume)

				bytesQuota, ok := subvolumeInfo.BytesQuota.(admin.ByteCount)
				if ok {
					ch <- prometheus.MustNewConstMetric(cephfsSubvolumeQuotaBytes, prometheus.GaugeValue, float64(bytesQuota), volume, group, subvolume)
				}
			}
		}
	}
}

func main() {
	flag.Parse()

	if *printVersion {
		fmt.Printf("%s\n", version)
		return
	}

	exporter := NewCephFSExporter()
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>CephFS Exporter</title></head>
			<body>
			<h1>CephFS Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
		`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
