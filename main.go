package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var configFile string
var quotaFile string
var debug bool

func init() {
	flag.StringVar(&configFile, "c", "", "config file")
	flag.StringVar(&quotaFile, "q", "", "quota file")
	flag.BoolVar(&debug, "d", false, "enable debug logging")
}

func main() {
	flag.Parse()
	debugLog("debug mode is enabled")

	err := loadConfig()
	if err != nil {
		debugLog("error loading config file")
		log.Fatal(err)
	}
	debugLog("config file loaded")

	debugLog("starting rgw-exporter")
	startRGWStatCollector()
	exporter := NewRGWExporter()
	prometheus.MustRegister(exporter)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/metrics/", promhttp.Handler())
	listenAddr := fmt.Sprintf("%s:%d", config.ListenIP, config.ListenPort)
	log.Printf("beginning to serve on %s:%d", config.ListenIP, config.ListenPort)

	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func debugLog(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}
