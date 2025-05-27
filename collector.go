package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

func startRGWStatCollector() {
	conn := getRGWConnection()
	tickerUsage := time.NewTicker(time.Duration(config.UsageCollectorInterval) * time.Second)
	tickerBuckets := time.NewTicker(time.Duration(config.BucketsCollectorInterval) * time.Second)
	tickerUsers := time.NewTicker(time.Duration(config.UsersCollectorInterval) * time.Second)
	tickerLc := time.NewTicker(time.Duration(config.LcCollectorInterval) * time.Second)

	// usage collector ticker
	go func() {
		debugLog("starting usage collector ticker")
		for ; ; <-tickerUsage.C {
			if isMaster() {
				collectUsage(conn, config.UsageSkipWithoutBucket)
			} else if usageMap != nil {
				debugLog("not master node: clearing usage statistics")
				usageMu.Lock()
				// usageMap = make(map[UsageKey]*UsageStats)
				usageMap = nil
				usageMu.Unlock()
				collectUsageDurationMu.Lock()
				collectUsageDuration = time.Duration(0)
				collectUsageDurationMu.Unlock()
			}
		}
	}()

	// buckets collector ticker
	go func() {
		debugLog("starting buckets collector ticker")
		for ; ; <-tickerBuckets.C {
			if isMaster() {
				collectBuckets(conn)
			} else if buckets != nil {
				debugLog("not master node: clearing buckets statistics")
				bucketsMu.Lock()
				buckets = nil
				bucketsMu.Unlock()
				collectBucketsDurationMu.Lock()
				collectBucketsDuration = time.Duration(0)
				collectBucketsDurationMu.Unlock()
			}
		}
	}()

	// users collector ticker
	go func() {
		if config.UsersCollectorEnable {
			debugLog("starting users collector ticker")
			for ; ; <-tickerUsers.C {
				if isMaster() {
					collectUsers(conn, config.UsersCollectorShowAllUsers)
				} else if users != nil {
					debugLog("not master node: clearing users statistics")
					usageMu.Lock()
					users = nil
					usageMu.Unlock()
					collectUsersDurationMu.Lock()
					collectUsersDuration = time.Duration(0)
					collectUsersDurationMu.Unlock()
				}
			}
		}
	}()

	// lc collector ticker
	go func() {
		if config.LcCollectorEnable {
			debugLog("starting lc collector ticker")
			for ; ; <-tickerLc.C {
				if isMaster() {
					collectBucketsLC(conn, config.Realm)
				} else if bucketsLcExpiration != nil {
					debugLog("not master node: clearing lc statistics")
					bucketsLcExpirationMu.Lock()
					bucketsLcExpiration = nil
					bucketsLcExpirationMu.Unlock()
					collectLcDurationMu.Lock()
					collectLcDuration = time.Duration(0)
					collectLcDurationMu.Unlock()
				}
			}
		}
	}()

	// tick every 10 seconds
	// if instance is master and data is missing, trigger collection
	go func() {
		// delay before starting ticker
		time.Sleep(60 * time.Second)
		debugLog("starting fast collector ticker")
		t := time.NewTicker(10 * time.Second)
		for ; ; <-t.C {
			if isMaster() {
				if usageMap == nil {
					debugLog("fast ticker usage collector started")
					collectUsage(conn, config.UsageSkipWithoutBucket)
				}
				if buckets == nil {
					debugLog("fast ticker buckets collector started")
					collectBuckets(conn)
				}
				if config.UsersCollectorEnable {
					if users == nil {
						debugLog("fast ticker users collector started")
						collectUsers(conn, config.UsersCollectorShowAllUsers)
					}
				}
				if config.LcCollectorEnable {
					if bucketsLcExpiration == nil {
						debugLog("fast ticker lc collector started")
						collectBucketsLC(conn, config.Realm)
					}
				}
			}
		}
	}()
}

func getRGWConnection() *rgw.API {
	// Verify SSL Certificate
	var tr *http.Transport
	if config.RGWConnectionCheckSSL {
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

func isMaster() bool {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
	}
	for _, addr := range addrList {
		if ip, ok := addr.(*net.IPNet); ok && ip.IP.String() == config.MasterIP {
			return true
		}
	}
	return false
}
