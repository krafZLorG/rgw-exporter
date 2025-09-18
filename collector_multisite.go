package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type MultisiteSyncStatus struct {
	MetadataLagSeconds int64 `json:"metadata_lag_seconds"`
	DataLagSeconds     int64 `json:"data_lag_seconds"`
}

var (
	multisiteStatus   *MultisiteSyncStatus
	multisiteStatusMu sync.Mutex
)
var (
	collectMultisiteStatusDuration   time.Duration
	collectMultisiteStatusDurationMu sync.Mutex
)

func collectMultisiteStatus(realm string) {
	debugLog("multisite sync status collector started")
	start := time.Now()
	// var curMultisiteSyncStatus []MultisiteSyncStatus
	curMultisiteSyncStatus, err := getMultisiteSyncStatus(realm)
	if err != nil {
		log.Printf("Error get multisite sync status: %v\n", err)
		return
	}

	multisiteStatusMu.Lock()
	multisiteStatus = curMultisiteSyncStatus
	multisiteStatusMu.Unlock()

	collectMultisiteStatusDurationMu.Lock()
	collectMultisiteStatusDuration = time.Since(start)
	collectMultisiteStatusDurationMu.Unlock()
	debugLog("multisite sync status collector finished in %s", time.Since(start))
}

func getMultisiteSyncStatus(realm string) (*MultisiteSyncStatus, error) {
	cmd := exec.Command("sudo", "radosgw-admin", "sync", "status", "--rgw-realm", realm, "--rgw-verify-ssl", "false")

	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error running radosgw-admin: %v\n", err)
		return nil, err
	}

	curMultisiteSyncStatus, err := parseMultisiteSyncStatus(out)
	if err != nil {
		log.Printf("Error parsing sync status: %v\n", err)
		return nil, err
	}
	debugLog("Metadata lag: %d seconds\n", curMultisiteSyncStatus.MetadataLagSeconds)
	debugLog("Data lag: %d seconds\n", curMultisiteSyncStatus.DataLagSeconds)

	return curMultisiteSyncStatus, nil
}

func parseMultisiteSyncStatus(output []byte) (*MultisiteSyncStatus, error) {
	status := &MultisiteSyncStatus{
		MetadataLagSeconds: 0,
		DataLagSeconds:     0,
	}

	var currentStr string
	var metaOldest, dataOldest string
	var metaCaughtUp, dataCaughtUp, metaMaster bool
	var currentSection string // "metadata" or "data"

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// current time
		if strings.HasPrefix(line, "current time") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentStr = parts[2]
			}
		}

		// section tracking
		if strings.HasPrefix(line, "metadata sync") {
			currentSection = "metadata"
			if strings.Contains(line, "no sync (zone is master)") {
				metaMaster = true
			}
		}
		if strings.HasPrefix(line, "data sync") {
			currentSection = "data"
		}

		// caught up flags
		if strings.Contains(line, "metadata is caught up with master") {
			metaCaughtUp = true
		}
		if strings.Contains(line, "data is caught up with source") {
			dataCaughtUp = true
		}

		// if failed to retriev sync info
		if strings.Contains(line, "failed to retrieve sync info") {
			switch currentSection {
			case "metadata":
				status.MetadataLagSeconds = -1
			case "data":
				status.DataLagSeconds = -1
			}
		}

		// oldest incremental change not applied
		if strings.Contains(line, "oldest incremental change not applied") {
			parts := strings.Fields(line)
			if len(parts) >= 6 {
				ts := parts[5]
				// strip fractional seconds
				if i := strings.Index(ts, "."); i != -1 {
					ts = ts[:i]
				}
				// assign based on section
				switch currentSection {
				case "metadata":
					metaOldest = ts
				case "data":
					dataOldest = ts
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// parse current time
	curTime, err := time.Parse(time.RFC3339, currentStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current time: %v", err)
	}

	// compute metadata lag
	if status.MetadataLagSeconds != -1 && !metaMaster && !metaCaughtUp && metaOldest != "" {
		oldTime, err := time.Parse("2006-01-02T15:04:05", metaOldest)
		if err == nil {
			status.MetadataLagSeconds = int64(curTime.Sub(oldTime).Seconds())
		}
	}

	// compute data lag
	if status.DataLagSeconds != -1 && !dataCaughtUp && dataOldest != "" {
		oldTime, err := time.Parse("2006-01-02T15:04:05", dataOldest)
		if err == nil {
			status.DataLagSeconds = int64(curTime.Sub(oldTime).Seconds())
		}
	}

	return status, nil
}
