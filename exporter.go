package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// pointers to prometheus descriptors for each metric
type RGWExporter struct {
	config Config
	//usage stat
	ops_total            *prometheus.Desc
	successful_ops_total *prometheus.Desc
	sent_bytes_total     *prometheus.Desc
	received_bytes_total *prometheus.Desc
	// bucket stat
	bucket_quota_enabled *prometheus.Desc
	bucket_quota_size    *prometheus.Desc
	bucket_quota_objects *prometheus.Desc
	bucket_size          *prometheus.Desc
	bucket_actual_size   *prometheus.Desc
	bucket_objects       *prometheus.Desc
	total_space          *prometheus.Desc
	// collector
	collector_buckets_duration_seconds *prometheus.Desc
	collector_usage_duration_seconds   *prometheus.Desc
}

// consructor for rgwCollector that initializes every decriptor
// and returns a pointer to the collector
func NewRGWExporter(config *Config) *RGWExporter {
	return &RGWExporter{
		config: *config,
		ops_total: prometheus.NewDesc("radosgw_usage_ops_total", "Number of requests",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		successful_ops_total: prometheus.NewDesc("radosgw_usage_successful_ops_total", "Number of successful requests",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		sent_bytes_total: prometheus.NewDesc("radosgw_usage_sent_bytes_total", "Bytes sent by the RGW",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		received_bytes_total: prometheus.NewDesc("radosgw_usage_received_bytes_total", "Bytes received by the RGW",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		bucket_quota_enabled: prometheus.NewDesc("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucket_quota_size: prometheus.NewDesc("radosgw_usage_bucket_quota_size", "Max allowed bucket size",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucket_quota_objects: prometheus.NewDesc("radosgw_usage_bucket_quota_objects", "Max allowed objects in_bucket",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucket_size: prometheus.NewDesc("radosgw_usage_bucket_size", "Bucket size bytes",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucket_actual_size: prometheus.NewDesc("radosgw_usage_bucket_actual_size", "Bucket size bytes",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucket_objects: prometheus.NewDesc("radosgw_usage_bucket_objects", "Bucket objecs count",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		total_space: prometheus.NewDesc("radosgw_usage_total_space", "Cluster total space TB",
			[]string{"cluster", "cluster_name", "realm", "realm_vrf"}, nil),
		collector_buckets_duration_seconds: prometheus.NewDesc("radosgw_usage_collector_buckets_duration_seconds", "Buckets collector duration time",
			[]string{"cluster", "realm"}, nil),
		collector_usage_duration_seconds: prometheus.NewDesc("radosgw_usage_collector_usage_duration_seconds", "Usage collector duration time",
			[]string{"cluster", "realm"}, nil),
	}
}

// collector must implement the Describe function that
// writes all descriptors to the prometheus desc channel
func (collector *RGWExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.ops_total
	ch <- collector.successful_ops_total
	ch <- collector.sent_bytes_total
	ch <- collector.received_bytes_total
	ch <- collector.bucket_quota_enabled
	ch <- collector.bucket_quota_size
	ch <- collector.bucket_quota_objects
	ch <- collector.bucket_size
	ch <- collector.bucket_actual_size
	ch <- collector.bucket_objects
	ch <- collector.total_space
	ch <- collector.collector_buckets_duration_seconds
	ch <- collector.collector_usage_duration_seconds
}

// collector must implement the Collect function
func (collector *RGWExporter) Collect(ch chan<- prometheus.Metric) {

	bucketsMu.Lock()
	defer bucketsMu.Unlock()

	for _, bucket := range buckets {

		// bucket_quota_enabled
		var quotaEnabled float64 = 0.0
		if *bucket.BucketQuota.Enabled {
			quotaEnabled = 1.0
		}
		ch <- prometheus.MustNewConstMetric(collector.bucket_quota_enabled, prometheus.GaugeValue, quotaEnabled,
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_quota_size
		ch <- prometheus.MustNewConstMetric(collector.bucket_quota_size, prometheus.GaugeValue, float64(*bucket.BucketQuota.MaxSize),
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_quota_objects
		ch <- prometheus.MustNewConstMetric(collector.bucket_quota_objects, prometheus.GaugeValue, float64(*bucket.BucketQuota.MaxObjects),
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_size
		var bucketSize float64 = 0.0
		if bucket.Usage.RgwMain.Size != nil {
			bucketSize = float64(*bucket.Usage.RgwMain.Size)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucket_size, prometheus.GaugeValue, bucketSize,
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_actual_size
		var bucketActualSize float64 = 0.0
		if bucket.Usage.RgwMain.SizeActual != nil {
			bucketActualSize = float64(*bucket.Usage.RgwMain.SizeActual)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucket_actual_size, prometheus.GaugeValue, bucketActualSize,
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_objects
		var bucketObjects float64 = 0.0
		if bucket.Usage.RgwMain.NumObjects != nil {
			bucketObjects = float64(*bucket.Usage.RgwMain.NumObjects)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucket_objects, prometheus.GaugeValue, bucketObjects,
			collector.config.ClusterFSID, collector.config.Realm, bucket.Tenant, bucket.Bucket)
	}

	usageMu.Lock()
	defer usageMu.Unlock()

	for key, stats := range usageMap {
		var user, tenant string
		userFullName := key.User
		if strings.Contains(userFullName, "$") {
			userSplit := strings.Split(userFullName, "$")
			tenant = userSplit[0]
			user = userSplit[1]
		} else {
			user = userFullName
			tenant = ""
		}
		ch <- prometheus.MustNewConstMetric(collector.sent_bytes_total, prometheus.CounterValue, float64(stats.BytesSent),
			collector.config.ClusterFSID, collector.config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.received_bytes_total, prometheus.CounterValue, float64(stats.BytesReceived),
			collector.config.ClusterFSID, collector.config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.ops_total, prometheus.CounterValue, float64(stats.Ops),
			collector.config.ClusterFSID, collector.config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.successful_ops_total, prometheus.CounterValue, float64(stats.SuccessfulOps),
			collector.config.ClusterFSID, collector.config.Realm, tenant, user, key.Bucket, key.Category)
	}

	// Summary metrics
	ch <- prometheus.MustNewConstMetric(collector.total_space, prometheus.GaugeValue, collector.config.ClusterSize,
		collector.config.ClusterFSID, collector.config.ClusterName, collector.config.Realm, collector.config.RealmVrf)
	ch <- prometheus.MustNewConstMetric(collector.collector_buckets_duration_seconds, prometheus.GaugeValue, collectBucketsDuration.Seconds(),
		collector.config.ClusterFSID, collector.config.Realm)
	ch <- prometheus.MustNewConstMetric(collector.collector_usage_duration_seconds, prometheus.GaugeValue, collectUsageDuration.Seconds(),
		collector.config.ClusterFSID, collector.config.Realm)
}
