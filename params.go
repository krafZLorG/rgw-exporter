package main

import (
	"flag"
	"fmt"
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
	UsageCollectorInterval     int     `yaml:"usage_collector_interval"`
	BucketsCollectorInterval   int     `yaml:"buckets_collector_interval"`
	RGWConnectionTimeout       int     `yaml:"rgw_connection_timeout"`
	StartDelay                 int     `yaml:"start_delay"`
	Insecure                   bool    `yaml:"insecure"`
	SkipWithoutBucket          bool    `yaml:"skip_without_bucket"`
	UsersCollectorEnable       bool    `yaml:"users_collector_enable"`
	UsersCollectorShowAllUsers bool    `yaml:"show_all_users"`
	UsersCollectorInterval     int     `yaml:"users_collector_interval"`
}

func loadConfig() (*Config, error) {
	config := &Config{}
	config.ListenIP = "127.0.0.1"
	config.ListenPort = 9240
	config.MasterIP = "127.0.0.1"
	config.UsageCollectorInterval = 30
	config.BucketsCollectorInterval = 300
	config.RGWConnectionTimeout = 10
	config.Insecure = false
	config.StartDelay = 30
	config.SkipWithoutBucket = false
	config.UsersCollectorEnable = false
	config.UsersCollectorShowAllUsers = false
	config.UsersCollectorInterval = 3600

	var configFile string
	flag.StringVar(&configFile, "c", "", "config file")
	flag.Parse()

	s, err := os.Stat(configFile)
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, fmt.Errorf("'%s' is a directory, not a normal file", configFile)
	}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
