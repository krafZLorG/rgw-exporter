package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

type BucketLcExpiration struct {
	Tenant string `yaml:"tenant"`
	Bucket string `yaml:"bucket"`
	Days   int    `yaml:"days"`
}

var (
	bucketsLcExpiration   []BucketLcExpiration
	bucketsLcExpirationMu sync.Mutex
)
var (
	collectLcDuration   time.Duration
	collectLcDurationMu sync.Mutex
)

func collectBucketsLC(conn *rgw.API, realm string) {
	debugLog("lc collector started")
	start := time.Now()
	var curBucketsLC []BucketLcExpiration

	buckets, err := conn.ListBuckets(context.Background())
	if err != nil {
		log.Println("lc collector Unable to get buckets list")
		return
	}
	debugLog("lc collector received buckets list from RGW: %v", time.Since(start))

	for _, bucket := range buckets {
		data := BucketLcExpiration{}
		if strings.Contains(bucket, "/") {
			userSplit := strings.Split(bucket, "/")
			data.Tenant = userSplit[0]
			data.Bucket = userSplit[1]
		} else {
			data.Tenant = ""
			data.Bucket = bucket
		}
		data.Days = GetBucketLcExpiration(bucket, realm)
		curBucketsLC = append(curBucketsLC, data)
	}

	bucketsLcExpirationMu.Lock()
	bucketsLcExpiration = curBucketsLC
	bucketsLcExpirationMu.Unlock()

	collectLcDurationMu.Lock()
	collectLcDuration = time.Since(start)
	collectLcDurationMu.Unlock()
	debugLog("lc collector finished in %s", time.Since(start))
}

func GetBucketLcExpiration(bucket string, realm string) int {
	start := time.Now()
	minExpiration := -1

	cmd := exec.Command("sudo", "radosgw-admin", "lc", "get", "--rgw-realm", realm, "--bucket", bucket)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("lc collector failed to get stdout pipe: %v", err)
		return -1
	}
	if err := cmd.Start(); err != nil {
		log.Printf("lc collector failed to start command: %v", err)
		return -1
	}
	data := bufio.NewReader(stdout)

	// radosgw-admin lc get command returns invalid JSON
	// The key may appear multiple times because LC may consist of multiple rules for the same prefix
	// This works because json.Decoder doesn't overwrite keys
	decoder := json.NewDecoder(data)

	tok, err := decoder.Token()
	if err != nil || tok != json.Delim('{') {
		debugLog("lc collector %v No Lifecycle or invalid JSON ", bucket)
		return -1
	}
	for decoder.More() {
		tok, _ := decoder.Token()
		key := tok.(string)

		if key == "prefix_map" {
			tok, _ = decoder.Token()
			if tok != json.Delim('{') {
				debugLog("lc collector %v Invalid JSON. Expected { for prefix_map", bucket)
				return -1
			}

			for decoder.More() {
				tok, _ := decoder.Token()
				prefix := tok.(string)

				var data map[string]interface{}
				err := decoder.Decode(&data)
				if err != nil {
					debugLog("lc collector %v Error decoding JSON: %v", bucket, err)
					return -1
				}

				if prefix == "" {
					if expirationVal, ok := data["expiration"]; ok {
						if expFloat, ok := expirationVal.(float64); ok {
							expiration := int(expFloat)
							if expiration != 0 {
								if minExpiration == -1 || expiration < minExpiration {
									minExpiration = expiration
								}
							}
						}
					}
				}
			}

		} else {
			// Skip other top-level keys
			var dummy interface{}
			err := decoder.Decode(&dummy)
			if err != nil {
				debugLog("lc collector %v Error decoding JSON: %v", bucket, err)
				return -1
			}
		}
	}
	debugLog("lc collector expiration %v %d %v", bucket, minExpiration, time.Since(start))

	return minExpiration
}
