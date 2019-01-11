package zlutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
)

func HttpPost(ctx context.Context, client *http.Client, url string, reqBody interface{}, respBody interface{}) (err error) {
	defer BeginSubsegment(&ctx)()

	entry := logrus.WithFields(logrus.Fields{
		"url":     url,
		"reqBody": reqBody,
	})

	reqBs, err := json.Marshal(&reqBody)
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	resp, err := ctxhttp.Post(ctx, client, url, "application/json", bytes.NewBuffer(reqBs))
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	entry.WithField("body", string(body)).Debug()

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(body, respBody); err != nil {
			entry.WithError(err).WithField("body", string(body)).Error()
			return
		}
		return nil
	} else {
		err = fmt.Errorf("status code is't 200")
		entry.WithError(err).WithField("body", string(body)).Error()
		return
	}
}

func HttpGet(ctx context.Context, client *http.Client, url string, respBody interface{}) (err error) {
	defer BeginSubsegment(&ctx)()

	entry := logrus.WithField("url", url)

	resp, err := ctxhttp.Get(ctx, client, url)
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	entry.WithField("body", string(body)).Debug()

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(body, respBody); err != nil {
			entry.WithError(err).WithField("body", string(body)).Error()
			return
		}
		return
	} else {
		err = fmt.Errorf("status code is't 200")
		entry.WithError(err).WithField("body", string(body)).Error()
		return
	}
}

type apiCode struct {
	Code    int         `json:"ret"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

// 成功
func RespSuccess(data interface{}) (int, apiCode) {
	logrus.WithFields(logrus.Fields{
		"stack":  nil, //这里的stack没意义，显式关闭
		"source": GetSource(2),
		"data":   data,
	}).Debug()
	return http.StatusOK, apiCode{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// 校验query参数失败
func RespErrorQueryParams(err error) (int, apiCode) {
	if err != nil {
		//当外部打印日志后，应当传err=nil，免得重复打日志
		logrus.WithError(err).WithFields(logrus.Fields{
			"stack":  nil, //这里的stack没意义，显式关闭
			"source": GetSource(2),
		}).Warn()
	}
	return http.StatusBadRequest, apiCode{
		Code:    1002,
		Message: "verify query params failed",
	}
}

// 服务器错误
func RespErrorServer() (int, apiCode) {
	return http.StatusInternalServerError, apiCode{
		Code:    1000,
		Message: "server error",
	}
}

// 请求失败
func RespErrorRequestFailed(err error) (int, apiCode) {
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"stack":  nil, //这里的stack没意义，显式关闭
			"source": GetSource(2),
		}).Warn()
	}
	return http.StatusBadRequest, apiCode{
		Code:    1001,
		Message: "request failed",
	}
}

// 校验post参数失败
func RespErrorPostParams(err error) (int, apiCode) {
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"stack":  nil, //这里的stack没意义，显式关闭
			"source": GetSource(2),
		}).Warn()
	}
	return http.StatusBadRequest, apiCode{
		Code:    1004,
		Message: "verify post params failed",
	}
}

// 校验header参数失败
func RespErrorHeaderParams(err error) (int, apiCode) {
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"stack":  nil, //这里的stack没意义，显式关闭
			"source": GetSource(2),
		}).Warn()
	}
	return http.StatusBadRequest, apiCode{
		Code:    1005,
		Message: "verify header params failed",
	}
}
