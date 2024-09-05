package socket

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func CreateServer(isIPv6 bool, port uint32, callback func(clientCtx *ClientContext, data []byte)) (error, *ServerContext) {
	networkType := "tcp"
	if isIPv6 {
		networkType = "tcp6"
	}
	listen, err := net.Listen(networkType, fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println("Create Server Error! err = ", err)
		return err, nil
	}
	ctx := &ServerContext{
		isIPv6, port, &listen, []*ClientContext{}, false,
	}
	go func() {
		ctx.isRunning = true
		for ctx.isRunning {
			conn, err := (*ctx.listener).Accept()
			if err != nil {
				log.Println("accept error.", err)
				continue
			}
			cCtx := ClientContext{isIPv6, conn.RemoteAddr().String(), 0, &conn, true, 0, sync.Mutex{}, sync.Mutex{}}
			ctx.clients = append(ctx.clients, &cCtx)
			cCtx.onDataReady(callback)
			cCtx.startHeartBeat()
		}
		for _, client := range ctx.clients {
			(*client.conn).Close()
		}
		ctx.clients = make([]*ClientContext, 0)
	}()
	ctx.clearDisConnectedClient()
	return nil, ctx
}

func ConnectServer(isIPv6 bool, address string, port uint32, callback func(clientCtx *ClientContext, data []byte)) (error, *ClientContext) {
	networkType := "tcp"
	if isIPv6 {
		networkType = "tcp6"
	}
	ctx := &ClientContext{}
	ctx.isIPv6 = isIPv6
	ctx.iPAddr = address
	ctx.targetPort = port
	ctx.isConnected = true
	ctx.heartBeatTimestamp = 0
	ctx.mutexWrite = sync.Mutex{}
	ctx.mutexRead = sync.Mutex{}
	if isIPv6 {
		conn, err := net.Dial(networkType, fmt.Sprintf("[%s]:%d", address, port))
		if err != nil {
			log.Println("Connect Server Error! err = ", err)
			return err, nil
		} else {
			ctx.conn = &conn
		}
	} else {
		conn, err := net.Dial(networkType, fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			log.Println("Connect Server Error! err = ", err)
			return err, nil
		} else {
			ctx.conn = &conn
		}
	}
	ctx.onDataReady(callback)
	ctx.startHeartBeat()
	return nil, ctx
}

func (ctx *ServerContext) BroadCastDataToClientDefault(data []byte) {
	ctx.BroadCastDataToClient(data, DataTypePayload)
}

func (ctx *ServerContext) BroadCastDataToClient(data []byte, _type uint32) {
	dataType := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataType, _type)
	dataSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataSize, uint32(len(data)))
	clients := ctx.clients
	sendData := append(dataType, dataSize...)
	sendData = append(sendData, data...)
	for _, client := range clients {
		go func() {
			if client.isConnected {
				client.write(sendData)
			}
		}()
	}
}

func (ctx *ServerContext) clearDisConnectedClient() {
	go func() {
		for ctx.isRunning {
			newClients := make([]*ClientContext, 0)
			oldClients := ctx.clients
			for _, client := range oldClients {
				if client.isConnected {
					newClients = append(newClients, client)
				}
			}
			ctx.clients = newClients
			time.Sleep(5 * time.Second)
		}
	}()
}

func (ctx *ServerContext) StopServer() {
	go func() {
		ctx.isRunning = false
		clients := ctx.clients
		for _, client := range clients {
			client.DisConnect()
		}
		(*ctx.listener).Close()
		ctx.listener = nil
		ctx.clients = make([]*ClientContext, 0)
	}()
}

func (ctx *ClientContext) SendDataToServerDefault(data []byte) {
	ctx.SendDataToServer(data, DataTypePayload)
}

func (ctx *ClientContext) SendDataToServer(data []byte, _type uint32) {
	go func() {
		dataType := make([]byte, 4)
		binary.LittleEndian.PutUint32(dataType, _type)
		dataSize := make([]byte, 4)
		binary.LittleEndian.PutUint32(dataSize, uint32(len(data)))
		sendData := append(dataType, dataSize...)
		sendData = append(sendData, data...)
		ctx.write(sendData)
	}()
}

func (ctx *ClientContext) onDataReady(callback func(clientCtx *ClientContext, data []byte)) {
	go func() {
		for ctx.isConnected {
			dataType := make([]byte, 4)
			dataSize := make([]byte, 4)
			ctx.mutexRead.Lock()
			(*ctx.conn).Read(dataType)
			(*ctx.conn).Read(dataSize)
			_type := binary.LittleEndian.Uint32(dataType)
			size := binary.LittleEndian.Uint32(dataSize)
			data := make([]byte, size)
			(*ctx.conn).Read(data)
			ctx.mutexRead.Unlock()
			switch _type {
			case DataTypePayload:
				callback(ctx, data)
				break
			case DataTypeHeartBeat:
				ctx.onHeartBeatData()
				break
			case DataTypeHeartBeatResult:
				ctx.onHeartBeatDataResult()
				break
			case DataTypeDisConnect:
				ctx.onDisConnect()
			default:
				log.Println("Unknown DataType ", _type)
			}
		}
	}()
}

func (ctx *ClientContext) startHeartBeat() {
	go func() {
		for ctx.isConnected {
			data := []byte(string(time.Now().Unix()))
			ctx.SendDataToServer(data, DataTypeHeartBeat)
			time.Sleep(10 * time.Second)
			if time.Now().Unix()-ctx.heartBeatTimestamp > 10 {
				ctx.isConnected = false
				(*ctx.conn).Close()
			}
		}
	}()
}

func (ctx *ClientContext) onHeartBeatData() {
	if ctx.isConnected {
		data := []byte(string(time.Now().Unix()))
		ctx.SendDataToServer(data, DataTypeHeartBeatResult)
	}
}

func (ctx *ClientContext) onHeartBeatDataResult() {
	ctx.heartBeatTimestamp = time.Now().Unix()
}

func (ctx *ClientContext) write(data []byte) error {
	ctx.mutexWrite.Lock()
	size := len(data)
	for size > 0 {
		n, err := (*ctx.conn).Write(data)
		if err != nil {
			ctx.mutexWrite.Unlock()
			log.Println("write data err = ", err)
			return err
		}
		size -= n
	}
	ctx.mutexWrite.Unlock()
	return nil
}

func (ctx *ClientContext) IsConnected() bool {
	return ctx.isConnected
}

func (ctx *ClientContext) onDisConnect() {
	ctx.isConnected = false
	(*ctx.conn).Close()
}

func (ctx *ClientContext) DisConnect() {
	data := []byte(string(time.Now().Unix()))
	ctx.SendDataToServer(data, DataTypeDisConnect)
	ctx.isConnected = false
	(*ctx.conn).Close()
	ctx.conn = nil
}
