package logger

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func InitDefaultMetric(projectName string) {
	//log次数,写日志很快所以没有计时
	defaultCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_log_total", projectName),
			Help: "Total Log counts",
		},
		[]string{"level"},
	)
	prometheus.MustRegister(
		defaultCounter,
	)
	MetricCounter = func(entry *logrus.Entry) prometheus.Counter {
		return defaultCounter.WithLabelValues(entry.Level.String())
	}
}

type fec func(entry *logrus.Entry) prometheus.Counter

var MetricCounter fec
