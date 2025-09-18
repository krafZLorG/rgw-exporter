package main

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type RGWExporter struct {
	// config Config
	//usage stat
	opsTotal           *prometheus.Desc
	successfulOpsTotal *prometheus.Desc
	sentBytesTotal     *prometheus.Desc
	receivedBytesTotal *prometheus.Desc
	// bucket stat
	bucketQuotaEnabled *prometheus.Desc
	bucketQuotaSize    *prometheus.Desc
	bucketQuotaObjects *prometheus.Desc
	bucketSize         *prometheus.Desc
	bucketActualSize   *prometheus.Desc
	bucketObjects      *prometheus.Desc
	bucketLcExpiration *prometheus.Desc
	userSuspended      *prometheus.Desc
	totalSpace         *prometheus.Desc
	// Multisite stat
	multisiteLagMetadata *prometheus.Desc
	multisiteLagData     *prometheus.Desc
	// collector
	collectorBucketsDurationSeconds         *prometheus.Desc
	collectorUsageDurationSeconds           *prometheus.Desc
	collectorUsersDurationSeconds           *prometheus.Desc
	collectorLcDurationSeconds              *prometheus.Desc
	collectorMultisiteStatusDurationSeconds *prometheus.Desc
}

// NewRGWExporter constructor for rgwCollector that initializes every descriptor
// and returns a pointer to the collector
func NewRGWExporter() *RGWExporter {
	return &RGWExporter{
		opsTotal: prometheus.NewDesc("radosgw_usage_ops_total", "Number of requests",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		successfulOpsTotal: prometheus.NewDesc("radosgw_usage_successful_ops_total", "Number of successful requests",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		sentBytesTotal: prometheus.NewDesc("radosgw_usage_sent_bytes_total", "Bytes sent by the RGW",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		receivedBytesTotal: prometheus.NewDesc("radosgw_usage_received_bytes_total", "Bytes received by the RGW",
			[]string{"cluster", "realm", "tenant", "user", "bucket", "category"}, nil),
		bucketQuotaEnabled: prometheus.NewDesc("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucketQuotaSize: prometheus.NewDesc("radosgw_usage_bucket_quota_size", "Max allowed bucket size",
			[]string{"cluster", "realm", "tenant", "bucket", "uid"}, nil),
		bucketQuotaObjects: prometheus.NewDesc("radosgw_usage_bucket_quota_objects", "Max allowed objects in_bucket",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucketSize: prometheus.NewDesc("radosgw_usage_bucket_size", "Bucket size bytes",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucketActualSize: prometheus.NewDesc("radosgw_usage_bucket_actual_size", "Bucket size bytes",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucketObjects: prometheus.NewDesc("radosgw_usage_bucket_objects", "Bucket objects count",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		bucketLcExpiration: prometheus.NewDesc("radosgw_usage_bucket_lc_expiration", "Expiration days for bucket lifecycle rules with no prefix",
			[]string{"cluster", "realm", "tenant", "bucket"}, nil),
		userSuspended: prometheus.NewDesc("radosgw_usage_user_suspended", "1 - suspended, 0 - active",
			[]string{"cluster", "realm", "tenant", "uid", "display_name"}, nil),
		totalSpace: prometheus.NewDesc("radosgw_usage_total_space", "Cluster total space TB",
			[]string{"cluster", "cluster_name", "realm", "realm_vrf"}, nil),
		multisiteLagMetadata: prometheus.NewDesc("radosgw_usage_multisite_metadata_lag", "Lag of multisite metadata sync in seconds (0 if caught up or master site).",
			[]string{"cluster", "cluster_name", "realm", "realm_vrf"}, nil),
		multisiteLagData: prometheus.NewDesc("radosgw_usage_multisite_data_lag", "Lag of multisite data sync in seconds (0 if caught up).",
			[]string{"cluster", "cluster_name", "realm", "realm_vrf"}, nil),
		collectorBucketsDurationSeconds: prometheus.NewDesc("radosgw_usage_collector_buckets_duration_seconds", "Buckets collector duration time",
			[]string{"cluster", "realm"}, nil),
		collectorUsageDurationSeconds: prometheus.NewDesc("radosgw_usage_collector_usage_duration_seconds", "Usage collector duration time",
			[]string{"cluster", "realm"}, nil),
		collectorUsersDurationSeconds: prometheus.NewDesc("radosgw_usage_collector_users_duration_seconds", "Users collector duration time",
			[]string{"cluster", "realm"}, nil),
		collectorLcDurationSeconds: prometheus.NewDesc("radosgw_usage_collector_lc_duration_seconds", "LC collector duration time",
			[]string{"cluster", "realm"}, nil),
		collectorMultisiteStatusDurationSeconds: prometheus.NewDesc("radosgw_usage_collector_multisite_status_duration_seconds", "Multisite status collector duration time",
			[]string{"cluster", "realm"}, nil),
	}
}

// Describe collector must implement the Describe function that
// writes all descriptors to the prometheus desc channel
func (collector *RGWExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.opsTotal
	ch <- collector.successfulOpsTotal
	ch <- collector.sentBytesTotal
	ch <- collector.receivedBytesTotal
	ch <- collector.bucketQuotaEnabled
	ch <- collector.bucketQuotaSize
	ch <- collector.bucketQuotaObjects
	ch <- collector.bucketSize
	ch <- collector.bucketActualSize
	ch <- collector.bucketObjects
	ch <- collector.bucketLcExpiration
	ch <- collector.userSuspended
	ch <- collector.totalSpace
	ch <- collector.multisiteLagMetadata
	ch <- collector.multisiteLagData
	ch <- collector.collectorBucketsDurationSeconds
	ch <- collector.collectorUsageDurationSeconds
	ch <- collector.collectorUsersDurationSeconds
	ch <- collector.collectorLcDurationSeconds
	ch <- collector.collectorMultisiteStatusDurationSeconds
}

// Collect collector must implement the Collect function
func (collector *RGWExporter) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	debugLog("exporter: collecting RGW metrics...")
	bucketsMu.Lock()
	defer bucketsMu.Unlock()

	for _, bucket := range buckets {
		// bucket_quota_enabled
		var quotaEnabled = 0.0
		if *bucket.BucketQuota.Enabled {
			quotaEnabled = 1.0
		}
		// bucket owner name
		var ownerUid = ""
		if strings.Contains(bucket.Owner, "$") {
			ownerUid = strings.Split(bucket.Owner, "$")[1]
		} else {
			ownerUid = bucket.Owner
		}

		ch <- prometheus.MustNewConstMetric(collector.bucketQuotaEnabled, prometheus.GaugeValue, quotaEnabled,
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_quota_size
		if !customBucketQuotaExist(bucket.Tenant, bucket.Bucket) {
			ch <- prometheus.MustNewConstMetric(collector.bucketQuotaSize, prometheus.GaugeValue, float64(*bucket.BucketQuota.MaxSize),
				config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket, ownerUid)
		}
		// bucket_quota_objects
		ch <- prometheus.MustNewConstMetric(collector.bucketQuotaObjects, prometheus.GaugeValue, float64(*bucket.BucketQuota.MaxObjects),
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_size
		var bucketSize = 0.0
		if bucket.Usage.RgwMain.Size != nil {
			bucketSize = float64(*bucket.Usage.RgwMain.Size)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucketSize, prometheus.GaugeValue, bucketSize,
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_actual_size
		var bucketActualSize = 0.0
		if bucket.Usage.RgwMain.SizeActual != nil {
			bucketActualSize = float64(*bucket.Usage.RgwMain.SizeActual)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucketActualSize, prometheus.GaugeValue, bucketActualSize,
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
		// bucket_objects
		var bucketObjects = 0.0
		if bucket.Usage.RgwMain.NumObjects != nil {
			bucketObjects = float64(*bucket.Usage.RgwMain.NumObjects)
		}
		ch <- prometheus.MustNewConstMetric(collector.bucketObjects, prometheus.GaugeValue, bucketObjects,
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
	}

	for _, bucket := range CustomQuotaBuckets {
		var ownerUid = ""
		ch <- prometheus.MustNewConstMetric(collector.bucketQuotaSize, prometheus.GaugeValue, float64(bucket.MaxSize),
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket, ownerUid)
	}

	for _, bucket := range bucketsLcExpiration {
		ch <- prometheus.MustNewConstMetric(collector.bucketLcExpiration, prometheus.GaugeValue, float64(bucket.Days),
			config.ClusterFSID, config.Realm, bucket.Tenant, bucket.Bucket)
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
		ch <- prometheus.MustNewConstMetric(collector.sentBytesTotal, prometheus.CounterValue, float64(stats.BytesSent),
			config.ClusterFSID, config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.receivedBytesTotal, prometheus.CounterValue, float64(stats.BytesReceived),
			config.ClusterFSID, config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.opsTotal, prometheus.CounterValue, float64(stats.Ops),
			config.ClusterFSID, config.Realm, tenant, user, key.Bucket, key.Category)
		ch <- prometheus.MustNewConstMetric(collector.successfulOpsTotal, prometheus.CounterValue, float64(stats.SuccessfulOps),
			config.ClusterFSID, config.Realm, tenant, user, key.Bucket, key.Category)
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	for _, user := range users {
		ch <- prometheus.MustNewConstMetric(collector.userSuspended, prometheus.GaugeValue, float64(user.Suspended),
			config.ClusterFSID, config.Realm, user.Tenant, user.UserId, user.DisplayName)
	}

	// Multisite metrics
	if multisiteStatus != nil {
		ch <- prometheus.MustNewConstMetric(collector.multisiteLagMetadata, prometheus.GaugeValue, float64(multisiteStatus.MetadataLagSeconds),
			config.ClusterFSID, config.ClusterName, config.Realm, config.RealmVrf)
		ch <- prometheus.MustNewConstMetric(collector.multisiteLagData, prometheus.GaugeValue, float64(multisiteStatus.DataLagSeconds),
			config.ClusterFSID, config.ClusterName, config.Realm, config.RealmVrf)
	}

	// Summary metrics
	ch <- prometheus.MustNewConstMetric(collector.totalSpace, prometheus.GaugeValue, config.ClusterSize,
		config.ClusterFSID, config.ClusterName, config.Realm, config.RealmVrf)
	ch <- prometheus.MustNewConstMetric(collector.collectorBucketsDurationSeconds, prometheus.GaugeValue, collectBucketsDuration.Seconds(),
		config.ClusterFSID, config.Realm)
	ch <- prometheus.MustNewConstMetric(collector.collectorUsageDurationSeconds, prometheus.GaugeValue, collectUsageDuration.Seconds(),
		config.ClusterFSID, config.Realm)
	ch <- prometheus.MustNewConstMetric(collector.collectorUsersDurationSeconds, prometheus.GaugeValue, collectUsersDuration.Seconds(),
		config.ClusterFSID, config.Realm)
	ch <- prometheus.MustNewConstMetric(collector.collectorLcDurationSeconds, prometheus.GaugeValue, collectLcDuration.Seconds(),
		config.ClusterFSID, config.Realm)
	ch <- prometheus.MustNewConstMetric(collector.collectorMultisiteStatusDurationSeconds, prometheus.GaugeValue, collectMultisiteStatusDuration.Seconds(),
		config.ClusterFSID, config.Realm)
	debugLog("exporter: finished in %v", time.Since(start))
}

func customBucketQuotaExist(tenant string, bucket string) bool {
	for _, b := range CustomQuotaBuckets {
		if tenant == b.Tenant && bucket == b.Bucket {
			return true
		}
	}
	return false
}
