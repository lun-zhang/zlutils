package guard

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"zlutils/metric"
)

func InitDefaultMetric(projectName string) {
	defaultCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_func_total", projectName),
			Help: "Total Func counts",
		},
		[]string{"func"},
	)
	defaultLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_func_latency_millisecond", projectName),
			Help:    "Func latency (millisecond)",
			Buckets: metric.HistoryBuckets,
		},
		[]string{"func"},
	)
	prometheus.MustRegister(
		defaultCounter,
		defaultLatency,
	)
	//TODO: 是否需要把函数返回的err带上？
	MetricCounter = func(name string) prometheus.Counter {
		return defaultCounter.WithLabelValues(name)
	}
	MetricLatency = func(name string) prometheus.Observer {
		return defaultLatency.WithLabelValues(name)
	}
}

type fnc func(name string) prometheus.Counter
type fno func(name string) prometheus.Observer

var (
	MetricCounter fnc
	MetricLatency fno
)
