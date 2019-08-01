package logger

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var defaultLogCounter *prometheus.CounterVec //log次数
//写日志很快所以没有计时

func InitDefaultMetric(projectName string) {
	defaultLogCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_log_total", projectName),
			Help: "Total Log counts",
		},
		[]string{"level"},
	)
	prometheus.MustRegister(
		defaultLogCounter,
	)
	DefaultLogCounter = func(entry *logrus.Entry) prometheus.Counter {
		return defaultLogCounter.WithLabelValues(entry.Level.String())
	}
}

type fec func(entry *logrus.Entry) prometheus.Counter

var DefaultLogCounter fec
