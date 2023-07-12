package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

var (
	buckets   []rgw.Bucket
	bucketsMu sync.Mutex
)
var (
	usage   rgw.Usage
	usageMu sync.Mutex
)
var (
	collectUsageDuration   time.Duration
	collectUsageDurationMu sync.Mutex
)
var (
	collectBucketsDuration   time.Duration
	collectBucketsDurationMu sync.Mutex
)

func startRGWStatCollector(config *Config) {
	conn := getRGWConnection(config)
	tickerUsage := time.NewTicker(time.Duration(config.UsageCollectorInterval) * time.Second)
	tickerBuckets := time.NewTicker(time.Duration(config.BucketsCollectorInterval) * time.Second)

	go func() {
		for ; ; <-tickerUsage.C {
			if isMaster(config.MasterIP) {
				collectUsage(conn)
			} else {
				usageMu.Lock()
				usage = rgw.Usage{}
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

func collectUsage(conn *rgw.API) {
	start := time.Now()

	today := time.Now().UTC().Format(time.DateOnly)
	curUsage, err := conn.GetUsage(context.Background(), rgw.Usage{ShowSummary: func() *bool { b := false; return &b }(), Start: today})
	if err != nil {
		log.Println("Unable to get usage stat")
		return
	}

	usageMu.Lock()
	// defer usageMu.Unlock()
	usage = curUsage
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
