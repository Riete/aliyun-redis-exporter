package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"log"
	"net/http"

	"github.com/riete/aliyun-redis-exporter/exporter"
)

const ListenPort string = "10003"

func main() {
	redis := exporter.RedisExporter{}
	redis.InitGauge()
	registry := prometheus.NewRegistry()
	registry.MustRegister(&redis)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", ListenPort), nil))
}
