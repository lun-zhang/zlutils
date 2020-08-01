package caller

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const unknown = "unknown"

var pj string

func Stack(skip int) (names []string) {
	for i := skip; ; i++ {
		s := Caller(i)
		if s == unknown {
			break
		}
		if pj != "" && !strings.Contains(s, pj) {
			continue //NOTE: 调用栈太长了，只打印服务内的
		}
		names = append(names, s)
	}
	return
}

func Caller(skip int) (name string) {
	name = unknown
	if pc, _, line, ok := runtime.Caller(skip); ok {
		name = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
	}
	return
}

func Init(projectName string) {
	pj = projectName
}

func DebugStack() (names []string) {
	for _, s := range strings.Split(string(debug.Stack()), "\n") {
		if pj != "" && !strings.Contains(s, pj) {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		names = append(names, s)
	}
	return
}
