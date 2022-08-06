package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ceph/go-ceph/cephfs/admin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "cephfs"

var (
	listenAddress = flag.String("web.listen-address", ":9939", "Address to listen on")

	labels = []string{"volume", "subvolume"}

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
)

type CephFSExporter struct{}

func NewCephFSExporter() *CephFSExporter {
	return &CephFSExporter{}
}

func (e *CephFSExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- cephfsSubvolumeUsedBytes
	ch <- cephfsSubvolumeQuotaBytes
}

func (e *CephFSExporter) Collect(ch chan<- prometheus.Metric) {
	fsa, err := admin.New()
	if err != nil {
		log.Fatal(err)
	}

	// Get CephFS volumes
	volumes, err := fsa.ListVolumes()
	if err != nil {
		log.Fatalf("cephfs volume list failed: %s", err)
	}

	for _, volume := range volumes {
		// Get Cephfs subvolumes
		subvolumes, err := fsa.ListSubVolumes(volume, "")
		if err != nil {
			log.Fatalf("cephfs subvolume list failed: %s", err)
		}

		for _, subvolume := range subvolumes {
			subvolumeInfo, err := fsa.SubVolumeInfo(volume, "", subvolume)
			if err != nil {
				log.Fatalf("cephfs subvolume info failed: %s", err)
			}

			ch <- prometheus.MustNewConstMetric(cephfsSubvolumeUsedBytes, prometheus.GaugeValue, float64(subvolumeInfo.BytesUsed), volume, subvolume)

			bytesQuota, ok := subvolumeInfo.BytesQuota.(admin.ByteCount)
			if ok {
				ch <- prometheus.MustNewConstMetric(cephfsSubvolumeQuotaBytes, prometheus.GaugeValue, float64(bytesQuota), volume, subvolume)
			}
		}
	}
}

func main() {
	flag.Parse()

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
