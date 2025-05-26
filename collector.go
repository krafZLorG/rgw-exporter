package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

var (
	buckets   []rgw.Bucket
	bucketsMu sync.Mutex
)
var (
	collectUsageDuration   time.Duration
	collectUsageDurationMu sync.Mutex
)
var (
	collectBucketsDuration   time.Duration
	collectBucketsDurationMu sync.Mutex
)
var (
	collectUsersDuration   time.Duration
	collectUsersDurationMu sync.Mutex
)
var (
	usageMap map[UsageKey]*UsageStats
	usageMu  sync.Mutex
)

var (
	users   []UserInfo
	usersMu sync.Mutex
)

type BucketLifecycle struct {
	Tenant string `yaml:"tenant"`
	Bucket string `yaml:"bucket"`
	Days   int    `yaml:"days"`
}

var (
	bucketsLifecycle   []BucketLifecycle
	bucketsLifecycleMu sync.Mutex
)
var (
	collectLifecycleDuration   time.Duration
	collectLifecycleDurationMu sync.Mutex
)

// Define the structure according to the JSON provided
type Category struct {
	Category      string `json:"category"`
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
	Ops           int64  `json:"ops"`
	SuccessfulOps int64  `json:"successful_ops"`
}

type Bucket struct {
	Bucket     string     `json:"bucket"`
	Time       string     `json:"time"`
	Epoch      int64      `json:"epoch"`
	Owner      string     `json:"owner"`
	Categories []Category `json:"categories"`
}

type UserUsage struct {
	User    string   `json:"user"`
	Buckets []Bucket `json:"buckets"`
}

type UserInfo struct {
	UserId      string `json:"user_id"`
	Tenant      string `url:"tenant"`
	DisplayName string `json:"display_name"`
	Suspended   int    `json:"suspended"`
}

// Key to identify unique combinations of user, bucket, owner, and category
type UsageKey struct {
	User     string
	Bucket   string
	Owner    string
	Category string
}

// Accumulated stats
type UsageStats struct {
	BytesSent     uint64
	BytesReceived uint64
	Ops           uint64
	SuccessfulOps uint64
}

func startRGWStatCollector(config *Config) {
	conn := getRGWConnection(config)
	tickerUsage := time.NewTicker(time.Duration(config.UsageCollectorInterval) * time.Second)
	tickerBuckets := time.NewTicker(time.Duration(config.BucketsCollectorInterval) * time.Second)
	tickerUsers := time.NewTicker(time.Duration(config.UsersCollectorInterval) * time.Second)
	tickerLifecycle := time.NewTicker(time.Duration(config.LifecycleCollectorInterval) * time.Second)

	go func() {
		for ; ; <-tickerUsage.C {
			if isMaster(config.MasterIP) {
				collectUsage(conn, config.SkipWithoutBucket)
			} else {
				usageMu.Lock()
				usageMap = make(map[UsageKey]*UsageStats)
				usageMu.Unlock()
			}
		}
	}()

	go func() {
		for ; ; <-tickerBuckets.C {
			if isMaster(config.MasterIP) {
				collectBuckets(conn)
			} else {
				bucketsMu.Lock()
				buckets = nil
				bucketsMu.Unlock()
			}
		}
	}()

	go func() {
		for ; ; <-tickerUsers.C {
			if isMaster(config.MasterIP) && config.UsersCollectorEnable {
				collectUsers(conn, config.UsersCollectorShowAllUsers)
			} else {
				usageMu.Lock()
				users = nil
				usageMu.Unlock()
			}
		}
	}()

	go func() {
		if config.LifecycleCollectorEnable {
			for ; ; <-tickerLifecycle.C {
				if isMaster(config.MasterIP) {
					collectBucketsLifecycle(conn, config.Realm)
				}
			}
		}
	}()

	// tick every 60 seconds and if isMaster and no data collect lifecycle
	go func() {
		if config.LifecycleCollectorEnable {
			// delay before starting ticker
			time.Sleep(60 * time.Second)
			t := time.NewTicker(60 * time.Second)
			for ; ; <-t.C {
				if isMaster(config.MasterIP) {
					if bucketsLifecycle == nil {
						collectBucketsLifecycle(conn, config.Realm)
					}
				} else {
					bucketsLifecycleMu.Lock()
					bucketsLifecycle = nil
					bucketsLifecycleMu.Unlock()
				}
			}
		}
	}()
}

func getRGWConnection(config *Config) *rgw.API {
	// Verify SSL Certificate
	var tr *http.Transport
	if config.Insecure {
		tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	} else {
		tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}}
	}

	conn, err := rgw.New(config.Endpoint, config.AccessKey, config.SecretKey,
		&http.Client{Timeout: time.Duration(config.RGWConnectionTimeout) * time.Second, Transport: tr})
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func collectUsage(conn *rgw.API, skipWithoutBucket bool) {
	start := time.Now()

	today := time.Now().UTC().Format(time.DateOnly)
	curUsage, err := conn.GetUsage(context.Background(), rgw.Usage{ShowSummary: func() *bool { b := false; return &b }(), Start: today})
	if err != nil {
		log.Println("Unable to get usage stat")
		return
	}

	usageMu.Lock()
	// defer usageMu.Unlock()
	usageMap = sumUsage(curUsage, skipWithoutBucket)
	usageMu.Unlock()

	collectUsageDurationMu.Lock()
	collectUsageDuration = time.Since(start)
	collectUsageDurationMu.Unlock()
}

func collectBuckets(conn *rgw.API) {
	start := time.Now()

	curBuckets, err := conn.ListBucketsWithStat(context.Background())
	if err != nil {
		log.Println("Unable to get bucket stat")
		return
	}

	bucketsMu.Lock()
	// defer bucketsMu.Unlock()
	buckets = curBuckets
	bucketsMu.Unlock()

	collectBucketsDurationMu.Lock()
	collectBucketsDuration = time.Since(start)
	collectBucketsDurationMu.Unlock()
}

func collectUsers(conn *rgw.API, showAllUsers bool) {
	start := time.Now()
	var curUsers []UserInfo

	curUsersList, err := conn.GetUsers(context.Background())
	if err != nil {
		log.Println("Unable to get users info")
		return
	}

	for _, v := range *curUsersList {
		curUser, err := conn.GetUser(context.Background(), rgw.User{ID: v})
		if err != nil {
			log.Println("Unable to get user info")
			return
		}
		user := UserInfo{curUser.ID, curUser.Tenant, curUser.DisplayName, *curUser.Suspended}
		if showAllUsers || (user.UserId == user.Tenant) {
			curUsers = append(curUsers, user)
		}
	}

	usersMu.Lock()
	users = curUsers
	usersMu.Unlock()

	collectUsersDurationMu.Lock()
	collectUsersDuration = time.Since(start)
	collectUsersDurationMu.Unlock()
}

func sumUsage(usage rgw.Usage, skipWithoutBucket bool) map[UsageKey]*UsageStats {

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
	return usageStatsMap
}

func collectBucketsLifecycle(conn *rgw.API, realm string) {
	start := time.Now()
	var curBucketsLifecycle []BucketLifecycle

	buckets, err := conn.ListBuckets(context.Background())
	if err != nil {
		log.Println("Lifecycle Collector: Unable to get buckets list")
		return
	}

	for _, bucket := range buckets {
		data := BucketLifecycle{}
		if strings.Contains(bucket, "/") {
			userSplit := strings.Split(bucket, "/")
			data.Tenant = userSplit[0]
			data.Bucket = userSplit[1]
		} else {
			data.Tenant = ""
			data.Bucket = bucket
		}
		data.Days = GetBucketLcExpiration(bucket, realm)
		curBucketsLifecycle = append(curBucketsLifecycle, data)
	}

	bucketsLifecycleMu.Lock()
	bucketsLifecycle = curBucketsLifecycle
	bucketsLifecycleMu.Unlock()

	collectLifecycleDurationMu.Lock()
	collectLifecycleDuration = time.Since(start)
	collectLifecycleDurationMu.Unlock()
}

func isMaster(vrrpIP string) bool {
	addr, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
	}
	for _, addr := range addr {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.String() == vrrpIP {
			return true
		}
	}
	return false
}
