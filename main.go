package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Println("Starting rgw-usage-exporter")

	config, err := loadConfig()
	if err != nil {
		log.Panic(err)
	}

	startRGWStatCollector(config)

	exporter := NewRGWExporter(config)
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())

	listenAddr := fmt.Sprintf("%s:%d", config.ListenIP, config.ListenPort)
	log.Printf("Beginning to serve on %s:%d", config.ListenIP, config.ListenPort)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
