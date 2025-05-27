package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AccessKey                  string  `yaml:"access_key"`
	SecretKey                  string  `yaml:"secret_key"`
	Endpoint                   string  `yaml:"endpoint"`
	ClusterFSID                string  `yaml:"cluster_fsid"`
	ClusterName                string  `yaml:"cluster_name"`
	ClusterSize                float64 `yaml:"cluster_size"`
	Realm                      string  `yaml:"realm"`
	RealmVrf                   string  `yaml:"realm_vrf"`
	ListenIP                   string  `yaml:"listen_ip"`
	ListenPort                 int     `yaml:"listen_port"`
	MasterIP                   string  `yaml:"master_ip"`
	RGWConnectionTimeout       int     `yaml:"rgw_connection_timeout"`
	RGWConnectionCheckSSL      bool    `yaml:"rgw_connection_check_ssl"`
	StartDelay                 int     `yaml:"start_delay"`
	UsageSkipWithoutBucket     bool    `yaml:"usage_skip_without_bucket"`
	UsageCollectorInterval     int     `yaml:"usage_collector_interval"`
	BucketsCollectorInterval   int     `yaml:"buckets_collector_interval"`
	UsersCollectorEnable       bool    `yaml:"users_collector_enable"`
	UsersCollectorShowAllUsers bool    `yaml:"users_collector_show_all_users"`
	UsersCollectorInterval     int     `yaml:"users_collector_interval"`
	LcCollectorEnable          bool    `yaml:"lc_collector_enable"`
	LcCollectorInterval        int     `yaml:"lc_collector_interval"`
}

var config Config

type CustomQuotaBucket struct {
	Tenant  string `yaml:"tenant"`
	Bucket  string `yaml:"bucket"`
	MaxSize int64  `yaml:"max_size"`
}

var CustomQuotaBuckets []CustomQuotaBucket

func loadConfig() error {
	configSetDefaults()

	debugLog("try to load config file: %s", configFile)
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close config file: %v\n", err)
		}
	}()
	debugLog("read config file: %s", configFile)
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&config); err != nil {
		return err
	}

	err = loadCustomQuotas()
	if err != nil {
		log.Println(err)
	}
	return nil
}

func loadCustomQuotas() error {
	if quotaFile == "" {
		quotaFile = "/etc/rgw-exporter/" + config.Realm + "_quotas.yaml"
	}

	debugLog("try to load customQuotas file: %s", quotaFile)
	file, err := os.Open(quotaFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close customQuotas file: %v\n", err)
		}
	}()

	debugLog("try to decode customQuotas")
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&CustomQuotaBuckets); err != nil {
		return err
	}
	return nil
}

func configSetDefaults() {
	config.AccessKey = "access"
	config.SecretKey = "secret"
	config.Endpoint = "http://127.0.0.1:8080"
	config.ClusterFSID = "00000000-0000-0000-0000-000000000000"
	config.ClusterName = "DEFAULT"
	config.ClusterSize = 1
	config.Realm = "default"
	config.RealmVrf = "DEFAULT"
	config.ListenIP = "127.0.0.1"
	config.ListenPort = 9240
	config.MasterIP = "127.0.0.1"
	config.RGWConnectionTimeout = 60
	config.RGWConnectionCheckSSL = false
	config.StartDelay = 30
	config.UsageSkipWithoutBucket = false
	config.UsageCollectorInterval = 30
	config.BucketsCollectorInterval = 300
	config.UsersCollectorEnable = false
	config.UsersCollectorShowAllUsers = false
	config.UsersCollectorInterval = 3600
	config.LcCollectorEnable = false
	config.LcCollectorInterval = 28800
}
