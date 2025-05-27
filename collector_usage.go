package main

import (
	"context"
	"log"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

type UserInfo struct {
	UserId      string `json:"user_id"`
	Tenant      string `url:"tenant"`
	DisplayName string `json:"display_name"`
	Suspended   int    `json:"suspended"`
}

type UsageKey struct {
	User     string
	Bucket   string
	Owner    string
	Category string
}

type UsageStats struct {
	BytesSent     uint64
	BytesReceived uint64
	Ops           uint64
	SuccessfulOps uint64
}

var (
	usageMap map[UsageKey]*UsageStats
	usageMu  sync.Mutex
)

var (
	collectUsageDuration   time.Duration
	collectUsageDurationMu sync.Mutex
)

func collectUsage(conn *rgw.API, skipWithoutBucket bool) {
	debugLog("usage collector started")
	start := time.Now()

	today := time.Now().UTC().Format(time.DateOnly)
	curUsage, err := conn.GetUsage(context.Background(), rgw.Usage{ShowSummary: func() *bool { b := false; return &b }(), Start: today})
	if err != nil {
		log.Println("unable to get usage statistics from rgw: ", err)
		return
	}
	debugLog("usage collector received usage statistics from RGW: %v", time.Since(start))
	curUsageMap := sumUsage(curUsage, skipWithoutBucket)

	usageMu.Lock()
	usageMap = curUsageMap
	usageMu.Unlock()

	collectUsageDurationMu.Lock()
	collectUsageDuration = time.Since(start)
	collectUsageDurationMu.Unlock()

	debugLog("usage collector finished in %s", time.Since(start))
}

func sumUsage(usage rgw.Usage, skipWithoutBucket bool) map[UsageKey]*UsageStats {
	start := time.Now()
	usageStatsMap := make(map[UsageKey]*UsageStats)

	// Iterate over the rgw.Usage entries
	for _, userUsage := range usage.Entries {
		for _, bucket := range userUsage.Buckets {
			if skipWithoutBucket {
				if bucket.Bucket == "" || bucket.Bucket == "-" {
					continue
				}
			}
			for _, category := range bucket.Categories {
				key := UsageKey{
					User:     userUsage.User,
					Bucket:   bucket.Bucket,
					Owner:    bucket.Owner,
					Category: category.Category,
				}

				if stats, exists := usageStatsMap[key]; !exists {
					usageStatsMap[key] = &UsageStats{
						BytesSent:     category.BytesSent,
						BytesReceived: category.BytesReceived,
						Ops:           category.Ops,
						SuccessfulOps: category.SuccessfulOps,
					}
				} else {
					stats.BytesSent += category.BytesSent
					stats.BytesReceived += category.BytesReceived
					stats.Ops += category.Ops
					stats.SuccessfulOps += category.SuccessfulOps
				}
			}
		}
	}
	debugLog("usage collector calculation finished in %v", time.Since(start))
	return usageStatsMap
}
