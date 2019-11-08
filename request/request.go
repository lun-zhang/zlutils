package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gosexy/to"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"zlutils/code"
	"zlutils/guard"
	zt "zlutils/time"
)

type MSI map[string]interface{}

//用于consul配置
type Config struct {
	Method string        `json:"method" validate:"oneof= GET POST PUT DELETE"`
	Url    string        `json:"url" validate:"url"`
	Client *ClientConfig `json:"client"`
}

type ClientConfig struct {
	Timeout   zt.Duration `json:"timeout"`
	Transport struct {
		MaxIdleConns        int         `json:"max_idle_conns"`
		MaxIdleConnsPerHost int         `json:"max_idle_conns_per_host"`
		IdleConnTimeout     zt.Duration `json:"idle_conn_timeout"`
		Dialer              struct {
			Timeout   zt.Duration `json:"timeout"`
			KeepAlive zt.Duration `json:"keep_alive"`
		} `json:"dialer"`
	} `json:"transport"`
}

func (m ClientConfig) GetClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        m.Transport.MaxIdleConns,
			MaxIdleConnsPerHost: m.Transport.MaxIdleConnsPerHost,
			IdleConnTimeout:     m.Transport.IdleConnTimeout.Duration,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := net.Dialer{
					Timeout:   m.Transport.Dialer.Timeout.Duration,
					KeepAlive: m.Transport.Dialer.KeepAlive.Duration,
				}
				return d.DialContext(ctx, network, addr)
			},
		},
		Timeout: m.Timeout.Duration,
	}
}

type Request struct {
	Config
	Query  MSI        //用于一次性设置
	query  url.Values //用于循环设置
	Header MSI
	Body   interface{}
}

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			//MaxIdleConns:        100,
			//MaxIdleConnsPerHost: 2,
			IdleConnTimeout: 5 * time.Minute,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: 10 * time.Minute,
				}
				return d.DialContext(ctx, network, addr)
			},
		},
		Timeout: 2 * time.Second,
	}
)

//用于循环设置
func (m *Request) AddQuery(k string, v interface{}) {
	if m.query == nil {
		m.query = url.Values{}
	}
	m.query.Add(k, to.String(v))
}

func (m Request) GetUrl(ctx context.Context) (string, error) {
	rawUrl, err := url.Parse(m.Url)
	if err != nil {
		logrus.WithContext(ctx).WithField("m", m).WithError(err).Error()
		return "", err
	}
	//NOTE: 将m.Url中的query、m.Query、m.query合并
	query := rawUrl.Query()
	for k, vs := range m.query {
		query[k] = append(query[k], vs...)
	}
	for k, v := range m.Query {
		query.Add(k, to.String(v))
	}

	rawUrl.RawQuery = query.Encode()
	return rawUrl.String(), nil
}

func (m Request) GetRequest(ctx context.Context) (request *http.Request, err error) {
	entry := logrus.WithContext(ctx).WithField("m", m)
	u, err := m.GetUrl(ctx)
	if err != nil {
		return
	}

	reqBodyBs, err := json.Marshal(m.Body)
	if err != nil {
		entry.WithError(err).Error()
		return
	}

	request, err = http.NewRequest(m.Method, u, bytes.NewBuffer(reqBodyBs))
	if err != nil {
		entry.WithError(err).Error()
		return
	}
	request.Header.Set("Content-Type", "application/json")
	for k, v := range m.Header {
		request.Header.Set(k, to.String(v))
	}
	entry.WithFields(logrus.Fields{
		"request_url":    request.URL.String(),
		"request_header": request.Header,
		"request_body":   m.Body,
	}).Debug()
	return
}

type RespBodyI interface {
	Check() error //自定义的错误码检查
}

//最常用的错误结构
type RespRet struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func (m RespRet) Check() error {
	if m.Ret != 0 {
		return fmt.Errorf("ret: %d != 0, msg: %s", m.Ret, m.Msg)
	}
	return nil
}

/*
错误码透传
如果客户端会将你服务A的msg作为toast弹出，而你又rpc调用了服务B，
那么你不应该把B的Code透传给客户端，而当返回server error
*/
type Pass code.Code

//已知
//我的rpc接口，返回的msg为 "simple_error: detail_error"
//罗浩的rpc接口，返回msg为 "simple_error. detail_error"
var splits = []string{": ", ". "} //simple_error中不可包含分隔符，目前看是没有的

//不是指针，不修改code值
func (m Pass) Check() error {
	if m.Ret == 0 {
		return nil
	}
	msg := m.Msg
	idx := -1
	var sp string
	for _, split := range splits {
		if i := strings.Index(msg, split); i >= 0 {
			idx = i
			msg = msg[:i]
			sp = split
		}
	}
	if idx < 0 {
		return code.Code(m)
	}
	e := m.Msg[idx+len(sp):]
	m.Msg = msg //将错误详情从msg剔出
	return code.Code(m).WithErrorf(e).WithSplit(sp)
}

//从logger里复制过来的
func tryGetJson(header http.Header, b []byte) (resp interface{}) {
	if strings.Contains(header.Get("Content-Type"), "application/json") {
		if er := json.Unmarshal(b, &resp); er == nil {
			return
		}
	}
	return string(b) //FIXME: 不会用非打印字符吧
}

func (m Request) Do(ctx context.Context, respBody RespBodyI) (err error) {
	defer guard.BeforeCtx(&ctx)(&err)
	entry := logrus.WithContext(ctx).WithField("m", m)

	request, err := m.GetRequest(ctx)
	if err != nil {
		return
	}
	entry = entry.WithFields(logrus.Fields{
		"request_url":    request.URL.String(),
		"request_header": request.Header,
	})
	client := defaultClient
	if m.Client != nil {
		client = m.Client.GetClient()
	}
	if seg := xray.GetSegment(ctx); seg != nil { //允许不传xray的ctx
		client = xray.Client(client)
	}
	resp, err := ctxhttp.Do(ctx, client, request)
	if err != nil { //超时
		//err = code.ServerErrRpc.WithError(err)
		entry.WithError(err).Error()
		return
	}
	defer resp.Body.Close()
	respBodyBs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		entry.WithError(err).Error()
		return
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		//影响性能，所以只有debug下才执行
		//另外在发生错误的时候也要执行
		entry = entry.WithField("response_body", tryGetJson(resp.Header, respBodyBs))
	}
	entry = entry.WithFields(logrus.Fields{
		"response_header": resp.Header,
		"StatusCode":      resp.StatusCode,
	})

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(respBodyBs, &respBody); err != nil {
			entry.WithField("response_body", tryGetJson(resp.Header, respBodyBs)).WithError(err).Error()
			return
		}
		if err = respBody.Check(); err != nil { //NOTE: ret!=0或者result!=ok等自定义的错误码
			//err = code.ServerErrRpc.WithError(err)
			entry.WithField("response_body", tryGetJson(resp.Header, respBodyBs)).WithError(err).Error()
			return
		}
		entry.Debug() //出错后会打err，因此不出错打debug
		return nil
	} else {
		err = fmt.Errorf("StatusCode %d != 200", resp.StatusCode)
		//err = code.ServerErrRpc.WithError(err)
		entry.WithField("response_body", tryGetJson(resp.Header, respBodyBs)).WithError(err).Error()
		return
	}
}
