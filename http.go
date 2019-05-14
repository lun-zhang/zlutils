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
	defer func() {
		if err != nil {
			err = CodeServerRpcErr.WithError(err)
		}
	}()

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
	defer func() {
		if err != nil {
			err = CodeServerRpcErr.WithError(err)
		}
	}()

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
