package xroad

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

func WrapError(err error) error {
	if err == nil {
		return err
	}
	f, file, line := caller(3)
	return errors.Wrap(err, fmt.Sprintf("%s:%d %s", file, line, f))
}

// return the caller's func name, file name, line number
func caller(skip int) (string, string, int) {
	pc := make([]uintptr, 1)
	runtime.Callers(skip, pc)
	f := runtime.FuncForPC(pc[0])

	parts := strings.Split(f.Name(), ".")
	funcName := parts[len(parts)-1]

	file, line := f.FileLine(pc[0])
	parts2 := strings.Split(file, "/")
	fileName := strings.Join([]string{parts2[len(parts2)-2], parts2[len(parts2)-1]}, "/")
	return funcName, fileName, line
}
