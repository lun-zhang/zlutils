package dtalk

import (
	"context"
	"fmt"
	"github.com/gosexy/to"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
	"zlutils/request"
)

var globalTokens map[logrus.Level]string

func Init(tokens map[logrus.Level]string) {
	globalTokens = tokens
}

var globalComTailLines []interface{}

//var globalComHeadLines []interface{}

//设置公参到末尾
func SetComTailLines(lines ...interface{}) {
	globalComTailLines = lines
}

//func SetComHeadLines(lines ...interface{}) {
//	globalComHeadLines = lines
//}

//异步，且捕获panic，不能影响主程序
//这样要求了必须传一个内容，而lines是可选
//本来想用map，因为经常指定k:v，但是顺序会变
func AsyncAlert(level logrus.Level, ctt string, lines ...interface{}) {
	//for _, line := range globalComHeadLines {
	//	ctt += "\n" + to.String(line)
	//}
	for _, line := range lines {
		ctt += "\n" + to.String(line)
	}
	for _, line := range globalComTailLines {
		ctt += "\n" + to.String(line)
	}
	go sendAlert(level, ctt, time.Now())
}

func sendAlert(level logrus.Level, ctt string, now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Warn("send alert panic", r)
		}
	}()

	var resp respAlert
	req := request.Request{
		Config: request.Config{
			Url:    "https://oapi.dingtalk.com/robot/send",
			Method: http.MethodPost,
		},
		Query: request.MSI{
			"access_token": globalTokens[level],
		},
		Body: request.MSI{
			"msgtype": "text",
			"text": request.MSI{
				"content": fmt.Sprintf("%s\n%s", ctt, now.UTC().Format(time.RFC3339)),
			},
		},
	}
	req.Do(context.Background(), &resp)
}

type respAlert struct {
	Errmsg  string `json:"errmsg"`
	Errcode int    `json:"errcode"`
}

func (m respAlert) Check() error {
	if m.Errcode != 0 {
		return fmt.Errorf("errcode: %d != 0, errmsg: %s", m.Errcode, m.Errmsg)
	}
	return nil
}
