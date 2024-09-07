package main

import (
	"IPv4ByIPv6/config"
	"IPv4ByIPv6/dispatch"
	"IPv4ByIPv6/eth"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error = nil
	err = checkPermission()
	if err != nil {
		panic(err)
	}
	if len(os.Args) > 2 {
		panic("wrong args count.")
	}
	var configCtx *config.ConfigContext = nil
	if len(os.Args) <= 1 {
		err, configCtx = config.GenConfigContext()
		if err == nil {
			log.Println("Master init success, join token is ", configCtx.GetJoinConfig())
		}
	} else {
		err, configCtx = config.GenConfigContextWithConfig(os.Args[1])
	}
	if err != nil || configCtx == nil {
		panic(err)
	}
	err, ethContext := eth.CreateTunDevice(configCtx.GetSegment(), configCtx.GetNumber())
	if err != nil {
		panic(err)
	}
	err, dispatchCtx := dispatch.CreateDispatchContext(configCtx, ethContext)
	if err != nil {
		panic(err)
	}
	err = dispatchCtx.StartDispatch()
	if err != nil {
		panic(err)
	}
	sig := make(chan os.Signal, 3)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGABRT, syscall.SIGHUP)
	<-sig
}
