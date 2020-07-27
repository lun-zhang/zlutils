package consul

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	Init(":8500", "tmp")

	GetAddress = func() (ip string, port int, err error) {
		return "localhost", 12345, nil
	}
	Register(RegisterConfig{
		//Stage:       "dev",
		ServiceName: "dev_hello",
		MetricsPath: "/hello/metrics",
		FailedFatal: true,
	})

	time.Sleep(time.Second * 5)

	DRegister()
}

func TestWatchChecks(t *testing.T) {
	WatchChecks("dev_hello", func(heathChecks []*api.HealthCheck) (err error) {
		fmt.Println(heathChecks)
		return
	})
	select {}
}

func TestServiceMaintenanceHandler(t *testing.T) {
	Init(":8500", "")
	GetAddress = func() (ip string, port int, err error) {
		return "localhost", 12345, nil
	}
	Register(RegisterConfig{
		ServiceName: "hello",
		MetricsPath: "/metrics",
	})
	go func() {
		c := make(chan os.Signal)
		//signal.Notify(c, syscall.SIGINT)
		signal.Notify(c, syscall.SIGTERM)
		<-c
		DRegister() //consul注销
	}()
	r := gin.New()
	h := ServiceMaintenanceHandler(func(in *MaintenanceCallbackIn) error {
		return fmt.Errorf("enable: %v", in.Enable)
		//return nil
	})
	r.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "abc")
	})
	r.POST("maintenance", func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	})
	r.Run(":12345")
}
