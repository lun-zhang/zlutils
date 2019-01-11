package zlutils

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

func JSONP(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		callback := r.URL.Query().Get("callback")
		if callback == "" {
			callback = r.URL.Query().Get("jsonp")
		}
		if callback == "" {
			next.ServeHTTP(w, r)
			return
		}

		wb := NewResponseBuffer(w)
		next.ServeHTTP(wb, r)

		if strings.Index(wb.Header().Get("Content-Type"), "/json") >= 0 {
			data := wb.Body.Bytes()
			wb.Body.Reset()

			wb.Body.Write([]byte(callback + "("))
			wb.Body.Write(data)
			wb.Body.Write([]byte(")"))

			wb.Header().Set("Content-Type", "application/javascript")
			wb.Header().Set("Content-Length", strconv.Itoa(wb.Body.Len()))
		}

		wb.Flush()
	}
	return http.HandlerFunc(fn)
}

type responseBuffer struct {
	Response http.ResponseWriter // the actual ResponseWriter to flush to
	Status   int                 // the HTTP response code from WriteHeader
	Body     *bytes.Buffer       // the response content body
	Flushed  bool
}

func NewResponseBuffer(w http.ResponseWriter) *responseBuffer {
	return &responseBuffer{
		Response: w, Status: 200, Body: &bytes.Buffer{},
	}
}

func (w *responseBuffer) Header() http.Header {
	return w.Response.Header() // use the actual response header
}

func (w *responseBuffer) Write(buf []byte) (int, error) {
	w.Body.Write(buf)
	return len(buf), nil
}

func (w *responseBuffer) WriteHeader(status int) {
	w.Status = status
}

func (w *responseBuffer) Flush() {
	if w.Flushed {
		return
	}
	w.Response.WriteHeader(w.Status)
	if w.Body.Len() > 0 {
		_, err := w.Response.Write(w.Body.Bytes())
		if err != nil {
			panic(err)
		}
		w.Body.Reset()
	}
	w.Flushed = true
}
