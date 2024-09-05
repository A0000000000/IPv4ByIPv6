package main

import (
	"os"
	"runtime"
)

func checkPermission() {
	currentOS := runtime.GOOS
	if currentOS == "linux" && os.Getuid() != 0 {
		panic("本程序需要root身份运行.")
	}
	if currentOS != "linux" {
		panic("暂不支持Linux以外的系统.")
	}

}
