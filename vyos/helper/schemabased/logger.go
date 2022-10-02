package schemabased

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

func logger(level string, msg string, values ...interface{}) {
	pc, file, line, _ := runtime.Caller(1)

	file_last_slash := strings.LastIndexByte(file, '/')
	if file_last_slash < 0 {
		file_last_slash = 0
	}

	func_name := runtime.FuncForPC(pc).Name()
	func_last_dot := strings.LastIndexByte(func_name, '.')
	if func_last_dot < 0 {
		func_last_dot = 0
	}

	logline := fmt.Sprintf("[%s] [%s:%s:%d] %s", level, file[file_last_slash+1:], func_name[func_last_dot+1:], line, msg)

	log.Printf(logline, values...)
}
