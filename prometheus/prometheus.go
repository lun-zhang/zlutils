package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"time"
	"zlutils/guard"
	"zlutils/request"
)

//目前只返回ecs的，允许用户自定义（也方便测试）
var GetAddress = func() (address string, err error) {
	for i := 1; i <= 32; i <<= 1 {
		time.Sleep(time.Duration(i) * time.Second)
		address, err = getAddressOnEcs()
		if err == nil {
			return
		}
	}
	err = fmt.Errorf("cant get address")
	logrus.WithError(err).Error()
	return "", err
}

func getAddressOnEcs() (address string, err error) {
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
	return fmt.Sprintf("%s:%d", metaData.HostPrivateIPv4Address, metaData.PortMappings[0].HostPort), nil
}

var reqCh = make(chan request.Request, 1)

func Unregister() (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer guard.BeforeCtx(&ctx)(&err)
	var resp request.RespRet

	select {
	case req, ok := <-reqCh:
		if ok {
			if err = req.Do(ctx, &resp); err != nil {
				return
			}
			logrus.WithField("req", req).Info("unregister ok")
		}
		//case <-time.After(timeout):
		//	err = fmt.Errorf("no req after %s timeout", timeout)
		//	logrus.WithError(err).Error()
	}
	return
}

func Register(requestConfig request.Config, jobName, metricsPath string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer guard.BeforeCtx(&ctx)(&err)
	defer func() {
		if err != nil {
			close(reqCh)
		}
	}()
	addr, err := GetAddress()
	if err != nil {
		return
	}

	var resp request.RespRet
	req := request.Request{
		Config: requestConfig,
		Body: request.MSI{
			"job_name":     jobName,
			"metrics_path": metricsPath,
			"add_targets": []string{
				addr,
			},
		},
	}
	if err = req.Do(ctx, &resp); err != nil {
		return
	}
	logrus.WithField("req", req).Info("register ok")
	reqCh <- request.Request{
		Config: requestConfig,
		Body: request.MSI{
			"job_name":     jobName,
			"metrics_path": metricsPath,
			"rem_targets": []string{
				addr,
			},
		},
	}
	return
}
