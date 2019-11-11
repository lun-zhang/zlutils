package mysql

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"zlutils/metric"
)

func InitDefaultMetric(projectName string) {
	defaultCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_mysql_total", projectName),
			Help: "Total Mysql counts",
		},
		[]string{"query"},
	)
	defaultLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_mysql_latency_millisecond", projectName),
			Help:    "Mysql latency (millisecond)",
			Buckets: metric.HistoryBuckets,
		},
		[]string{"query"},
	)
	prometheus.MustRegister(
		defaultCounter,
		defaultLatency,
	)
	MetricCounter = func(query string, args ...interface{}) prometheus.Counter {
		return defaultCounter.WithLabelValues(getSampleQuery(query))
	}
	MetricLatency = func(query string, args ...interface{}) prometheus.Observer {
		return defaultLatency.WithLabelValues(getSampleQuery(query))
	}
}

func getSampleQuery(query string) string {
	return strings.Replace(query, "?,", "", -1) //TODO: 改成更好的做法，把IN(?,...)替换成IN(...)
}

var (
	MetricCounter func(query string, args ...interface{}) prometheus.Counter
	MetricLatency func(query string, args ...interface{}) prometheus.Observer
)
