package main

import (
	"context"
	"log"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

var (
	buckets   []rgw.Bucket
	bucketsMu sync.Mutex
)
var (
	collectBucketsDuration   time.Duration
	collectBucketsDurationMu sync.Mutex
)

func collectBuckets(conn *rgw.API) {
	debugLog("buckets collector started")
	start := time.Now()

	curBuckets, err := conn.ListBucketsWithStat(context.Background())
	if err != nil {
		log.Println("unable to get buckets with stat")
		return
	}
	debugLog("buckets collector received %v buckets", len(curBuckets))

	bucketsMu.Lock()
	buckets = curBuckets
	bucketsMu.Unlock()

	collectBucketsDurationMu.Lock()
	collectBucketsDuration = time.Since(start)
	collectBucketsDurationMu.Unlock()
	debugLog("buckets collector finished in %s", collectBucketsDuration)
}
