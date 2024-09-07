package dispatch

import (
	"IPv4ByIPv6/config"
	"IPv4ByIPv6/eth"
	"IPv4ByIPv6/socket"
	"errors"
	"log"
	"sync"
)

func CreateDispatchContext(configCtx *config.ConfigContext, ethCtx *eth.EthContext) (error, *DispatchContext) {
	if configCtx == nil || ethCtx == nil {
		return errors.New("params cannot be nil!"), nil
	}
	return nil, &DispatchContext{
		configCtx:      configCtx,
		ethCtx:         ethCtx,
		mutexWrite:     sync.Mutex{},
		mutexRead:      sync.Mutex{},
		clientCtxCache: make(map[uint32]*socket.ClientContext),
	}
}

func (ctx *DispatchContext) StartDispatch() error {
	// 创建TCP6 Server
	err, socketCtx := socket.CreateServer(true, config.ListenPort, func(clientCtx *socket.ClientContext, data []byte) {
		// 写回数据，即远端设备向本地设备写回的数据
		ctx.writeToTun(data)
	})
	if err != nil {
		return err
	}
	ctx.socketCtx = socketCtx
	ctx.readFromTun()
	return nil
}

func (ctx *DispatchContext) writeToTun(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty data ignore.")
	}
	ctx.mutexWrite.Lock()
	size := len(data)
	for size > 0 {
		n, err := ctx.ethCtx.Write(data)
		if err != nil {
			ctx.mutexWrite.Unlock()
			return err
		}
		size -= n
	}
	ctx.mutexWrite.Unlock()
	return nil
}

func (ctx *DispatchContext) readFromTun() {
	go func() {
		for {
			ctx.mutexRead.Lock()
			// 由于以太网帧的MTU为1500，所以IP报文超过1500时，会进行分片，这里使用2000字节的buff，足够存储一次http的报文
			data := make([]byte, 2000)
			n, err := ctx.ethCtx.Read(data)
			data = data[:n]
			if err != nil {
				// 出错忽略
				continue
			}
			// 获取IPv4 头大小及整体大小
			//headerSize := uint32(data[0]&0xF) * 4
			//log.Println("headerSize = ", headerSize)
			//payloadSize := uint32(data[2])<<8 + uint32(data[3]) - headerSize
			//log.Println("headerSize = ", headerSize, ", payloadSize = ", payloadSize)
			if (data[0] >> 4) == 0x4 {
				// IPv4 报文
				ipv4 := data[16:20]
				clientCtx, has := ctx.clientCtxCache[uint32(ipv4[3])]
				if has && clientCtx != nil && clientCtx.IsConnected() {
					clientCtx.SendDataToServerDefault(data)
				} else {
					err, cfg, has := ctx.configCtx.QueryIPv6Address(uint32(ipv4[3]))
					if err == nil && has {
						err, clientCtx := socket.ConnectServer(true, cfg.GetIPv6Address(), cfg.GetIPv6Port(), func(clientCtx *socket.ClientContext, data []byte) {
							ctx.writeToTun(data)
						})
						if err == nil {
							clientCtx.SendDataToServerDefault(data)
							ctx.clientCtxCache[uint32(ipv4[3])] = clientCtx
						}
					} else {
						log.Println("query error. err = ", err, ", has = ", has)
					}
				}
			} else {
				// 其他 报文
				// ignore
			}
			ctx.mutexRead.Unlock()
		}
	}()
}
