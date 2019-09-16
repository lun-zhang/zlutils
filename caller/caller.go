package caller

import (
	"fmt"
	"regexp"
	"runtime"
)

const unknown = "unknown"

var pr *regexp.Regexp

func Stack(skip int) (names []string) {
	for i := skip; ; i++ {
		s := Caller(i)
		if s == unknown {
			break
		}
		if pr != nil && !pr.MatchString(s) {
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
	pr = regexp.MustCompile(projectName)
}
