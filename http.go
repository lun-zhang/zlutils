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
	entry = entry.WithField("body", string(body))
	entry.Debug()

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(body, respBody); err != nil {
			entry.WithError(err).Error()
			return
		}
		return nil
	} else {
		err = fmt.Errorf("StatusCode %d != 200", resp.StatusCode)
		entry.WithError(err).Error()
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
	entry = entry.WithField("body", string(body))
	entry.Debug()

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(body, respBody); err != nil {
			entry.WithError(err).Error()
			return
		}
		return nil
	} else {
		err = fmt.Errorf("StatusCode %d != 200", resp.StatusCode)
		entry.WithError(err).Error()
		return
	}
}

type ApiCode struct {
	Code    int         `json:"ret"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

//强行加
func (code ApiCode) AppendMsgForce(msg string) ApiCode {
	if msg != "" {
		code.Message = fmt.Sprintf("%s: %s", code.Message, msg)
	}
	return code
}

//只有debug模式才行
func (code ApiCode) AppendMsgDebug(msg string) ApiCode {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		return code.AppendMsgForce(msg)
	}
	return code
}

// 成功
func RespSuccess(data interface{}) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    0,
		Message: "success",
		Data:    data,
	}, "")
}

/*
ret统一，方便prometheus统计
正确:			0
客户端参数错误:	40xx
客户端逻辑错误:	41xx
服务器错误:		5xxx
*/

func Resp(code ApiCode, msg string) (int, ApiCode) {
	return http.StatusOK, code.AppendMsgDebug(msg)
}

// 请求失败
func RespErrorRequestFailed(msg string) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    4001,
		Message: "request failed",
	}, msg)
}

// 校验query参数失败
func RespErrorQueryParams(msg string) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    4002,
		Message: "verify query params failed",
	}, msg)
}

// 校验post参数失败
func RespErrorPostParams(msg string) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    4004,
		Message: "verify post params failed",
	}, msg)
}

// 校验header参数失败
func RespErrorHeaderParams(msg string) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    4005,
		Message: "verify header params failed",
	}, msg)
}

// 服务器错误
func RespErrorServer(msg string) (int, ApiCode) {
	return Resp(ApiCode{
		Code:    5000,
		Message: "server error",
	}, msg)
}
