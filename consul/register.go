package consul

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

//目前只返回ecs的，可手动修改方便调试
var GetAddress = func() (ip string, port int, err error) {
	for i := 1; i <= 32; i <<= 1 {
		time.Sleep(time.Duration(i) * time.Second)
		ip, port, err = getAddressOnEcs()
		if err == nil {
			return
		}
	}
	err = fmt.Errorf("cant get address")
	logrus.WithError(err).Error()
	return "", 0, err
}

func getAddressOnEcs() (ip string, port int, err error) {
	ecsMeta := os.Getenv("ECS_CONTAINER_METADATA_FILE")
	if ecsMeta == "" {
		err = fmt.Errorf("ecsMeta is empty")
		logrus.WithError(err).Error()
		return
	}
	entry := logrus.WithField("ecsMeta", ecsMeta)
	ecsData, err := ioutil.ReadFile(ecsMeta)
	if err != nil {
		entry.WithError(err).Warn("read ecs meta file failed")
		return
	}
	entry = entry.WithField("ecsData", string(ecsData))

	var metaData struct {
		PortMappings []struct {
			HostPort int `json:"HostPort"`
		} `json:"PortMappings"`
		HostPrivateIPv4Address string `json:"HostPrivateIPv4Address"`
		MetadataFileStatus     string `json:"MetadataFileStatus"`
	}
	if err = json.Unmarshal(ecsData, &metaData); err != nil {
		entry.WithError(err).Error("decode meta json failed")
		return
	}
	entry = entry.WithField("metaData", metaData)
	if metaData.MetadataFileStatus != "READY" || len(metaData.PortMappings) == 0 {
		err = fmt.Errorf("hasn't ready")
		entry.WithError(err).Error()
		return
	}
	entry.Debug()
	return metaData.HostPrivateIPv4Address, metaData.PortMappings[0].HostPort, nil
}

type RegisterConfig struct {
	//Stage       string `json:"env" validate:"oneof=local dev test prod"`
	ServiceName                    string `json:"service_name" validate:"required"`
	MetricsPath                    string `json:"metrics_path" validate:"required"`
	Interval                       string `json:"interval"`                          //检查间隔，默认5s
	Timeout                        string `json:"timeout"`                           //超时时间，默认3s
	FailedFatal                    bool   `json:"failed_fatal"`                      //true:注册失败则fatal
	DeregisterCriticalServiceAfter string `json:"deregister_critical_service_after"` //多久之后注销
}

var serviceIdCh = make(chan string, 1)
var serviceId string
var serviceName string

var registration *api.AgentServiceRegistration

func Register(config RegisterConfig) {
	entry := logrus.WithField("config", config)
	var err error
	defer func() {
		if err != nil {
			entry = entry.WithError(err)
			if config.FailedFatal {
				entry.Fatalf("服务注册失败，终止服务")
			} else {
				entry.Info("服务注册失败，继续服务")
			}
		}
	}()
	if err = vali.Struct(config); err != nil {
		return
	}

	ip, port, err := GetAddress()
	if err != nil {
		return
	}
	entry = entry.WithFields(logrus.Fields{
		"ip":   ip,
		"port": port,
	})
	if config.Interval == "" {
		config.Interval = "5s"
	}
	if config.Timeout == "" {
		config.Timeout = "3s"
	}
	address := fmt.Sprintf("%s:%d", ip, port)
	registration = &api.AgentServiceRegistration{
		Name:    config.ServiceName,
		ID:      address,
		Port:    port,
		Address: ip,
		Check: &api.AgentServiceCheck{
			Interval:                       config.Interval,
			Timeout:                        config.Timeout,
			Method:                         http.MethodGet,
			HTTP:                           fmt.Sprintf("http://%s%s", address, config.MetricsPath),
			DeregisterCriticalServiceAfter: config.DeregisterCriticalServiceAfter,
			Status:                         api.HealthPassing,
		},
	}
	err = Client.Agent().ServiceRegister(registration)
	if err != nil {
		return
	}
	serviceId = address
	serviceName = config.ServiceName
	serviceIdCh <- address
	entry.Infof("服务注册成功")
}

func DRegister() {
	serviceId := <-serviceIdCh
	entry := logrus.WithField("serviceId", serviceId)
	if err := Client.Agent().ServiceDeregister(serviceId); err != nil {
		entry.WithError(err).Error("服务注销失败")
		return
	}
	entry.Info("服务注销成功")
}

type WatchChecksCallback func(heathChecks []*api.HealthCheck) (err error)

// watch check
func WatchChecks(serviceName string, callback WatchChecksCallback) (err error) {
	plan, err := watch.Parse(map[string]interface{}{
		"type":    "checks",
		"service": serviceName,
	})
	if nil != err {
		logrus.Errorf("parse watch checks params error: %s", err)
		return
	}

	plan.HybridHandler = func(blockVal watch.BlockingParamVal, val interface{}) {
		healthChecks, ok := val.([]*api.HealthCheck)
		if !ok {
			return
		}

		errCallback := callback(healthChecks)
		if errCallback != nil {
			logrus.Errorf("watch checks callback error: %s", err)
			return
		}
	}

	go func() {
		errRun := plan.Run(Address)
		if nil != errRun {
			logrus.Errorf("error: %s", err)
		}
	}()
	return
}

// watch service
type WatchServiceCallback func(entries []*api.ServiceEntry) (err error)

// watch service
func WatchService(
	serviceName string,
	callback WatchServiceCallback,
) (err error) {
	plan, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	})
	if nil != err {
		logrus.Errorf("parse watch checks params error: %s", err)
		return
	}

	plan.HybridHandler = func(blockVal watch.BlockingParamVal, val interface{}) {
		serviceEntries, ok := val.([]*api.ServiceEntry)
		if !ok {
			return
		}

		errCallback := callback(serviceEntries)
		if errCallback != nil {
			logrus.Errorf("do watch service callback error: %s", err)
			return
		}
	}

	go func() {
		errRun := plan.Run(Address)
		if nil != errRun {
			logrus.Errorf("error: %s", err)
		}
	}()

	return
}

// 设置服务的维护状态
func setServiceMaintenanceMode(
	serviceId string,
	isMaintenance bool,
	enableReason map[string]interface{},
) (err error) {
	if isMaintenance {
		bReason, errM := json.Marshal(enableReason)
		err = errM
		if err != nil {
			logrus.Errorf("marshal error: %s", err)
			return
		}

		err = Client.Agent().EnableServiceMaintenance(serviceId, string(bReason))
		if err != nil {
			logrus.Errorf("enable service maintenance error: %s", err)
			return
		}

	} else {
		err = Client.Agent().DisableServiceMaintenance(serviceId)
		if err != nil {
			logrus.Errorf("disable service maintenance error: %s", err)
			return
		}
	}

	return
}

type MaintenanceCallbackIn struct {
	Enable       bool                   `json:"enable"` //true服务可用
	EnableReason map[string]interface{} `json:"enable_reason"`
}

//不与code关联
func ServiceMaintenanceHandler(callback func(*MaintenanceCallbackIn) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody MaintenanceCallbackIn
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("client err: " + err.Error()))
			return
		}

		if serviceId == "" || serviceName == "" || registration == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server err: serviceId or serviceName or registration is empty"))
			return
		}
		if !reqBody.Enable {
			registration.Name = serviceName + "_limited"
		} else {
			registration.Name = serviceName + "_active"
		}

		if err := Client.Agent().ServiceRegister(registration); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server err, rename failed: " + err.Error()))
			return
		}

		if err := setServiceMaintenanceMode(serviceId, !reqBody.Enable, reqBody.EnableReason); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server err: " + err.Error()))
			return
		}

		if err := callback(&reqBody); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server err, callback failed: " + err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("success %s %s", serviceId, registration.Name)))
	})
}
