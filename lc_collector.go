package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
)

func GetBucketLcExpiration(bucket string, realm string) int {
	minExpiration := -1

	// f := bucket + ".dat"
	// data, _ := os.Open(f)

	cmd := exec.Command("radosgw-admin", "--rgw-realm", realm, "lc", "get", "--bucket", bucket)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("LC failed to get stdout pipe: %v", err)
		return -1
	}
	if err := cmd.Start(); err != nil {
		log.Printf("LC failed to start command: %v", err)
		return -1
	}
	data := bufio.NewReader(stdout)

	// radosgw-admin lc get command returns invalid JSON
	// The key may appear multiple times because LC may consist of multiple rules for the same prefix
	// This works because json.Decoder doesn't overwrite keys
	decoder := json.NewDecoder(data)

	tok, err := decoder.Token()
	if err != nil || tok != json.Delim('{') {
		// todo debug
		// log.Printf("%v No Lifecycle or invalid JSON ", bucket)
		return -1
	}
	for decoder.More() {
		tok, _ := decoder.Token()
		key := tok.(string)

		if key == "prefix_map" {
			tok, _ = decoder.Token()
			if tok != json.Delim('{') {
				log.Printf("%v Invalid JSON. Expected { for prefix_map", bucket)
				return -1
			}

			for decoder.More() {
				tok, _ := decoder.Token()
				prefix := tok.(string)

				var data map[string]interface{}
				decoder.Decode(&data)

				if prefix == "" {
					if expirationVal, ok := data["expiration"]; ok {
						if expFloat, ok := expirationVal.(float64); ok {
							expiration := int(expFloat)
							if minExpiration == -1 || expiration < minExpiration {
								minExpiration = expiration
							}
						}
					}
				}
			}

		} else {
			// Skip other top-level keys
			var dummy interface{}
			decoder.Decode(&dummy)
		}
	}
	// todo debug
	// log.Printf("%v Minimum expiration for prefix \"\": %d", bucket, minExpiration)

	return minExpiration
}
