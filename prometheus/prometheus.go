package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
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
	f, err := os.Open(ecsMeta)
	if err != nil {
		logrus.WithError(err).Warn("read ecs meta file failed")
		return
	}
	defer f.Close()

	var metaData struct {
		PortMappings []struct {
			HostPort int `json:"HostPort"`
		} `json:"PortMappings"`
		HostPrivateIPv4Address string `json:"HostPrivateIPv4Address"`
		MetadataFileStatus     string `json:"MetadataFileStatus"`
	}
	if err = json.NewDecoder(f).Decode(metaData); err != nil {
		logrus.WithError(err).Error("decode meta json failed")
		return
	}
	if metaData.MetadataFileStatus != "READY" || len(metaData.PortMappings) == 0 {
		err = fmt.Errorf("hasn't ready")
		logrus.WithError(err).Error()
		return
	}
	return fmt.Sprintf("%s:%d", metaData.HostPrivateIPv4Address, metaData.PortMappings[0].HostPort), nil
}

func Register(requestConfig request.Config, jobName, metricsPath string) (unregister func(), err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer guard.BeforeCtx(&ctx)(&err)
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
	return func() {
		var err error
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer guard.BeforeCtx(&ctx)(&err)
		var resp request.RespRet
		req := request.Request{
			Config: requestConfig,
			Body: request.MSI{
				"job_name":     jobName,
				"metrics_path": metricsPath,
				"rem_targets": []string{
					addr,
				},
			},
		}
		if err = req.Do(ctx, &resp); err != nil {
			return
		}
	}, nil
}
