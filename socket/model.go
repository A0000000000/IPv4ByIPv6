package socket

import (
	"net"
	"sync"
)

const DataTypePayload uint32 = 0
const DataTypeHeartBeat uint32 = 1
const DataTypeHeartBeatResult uint32 = 2
const DataTypeDisConnect uint32 = 3

type ServerContext struct {
	isIPv6    bool
	port      uint32
	listener  *net.Listener
	clients   []*ClientContext
	isRunning bool
}

type ClientContext struct {
	isIPv6             bool
	iPAddr             string
	targetPort         uint32
	conn               *net.Conn
	isConnected        bool
	heartBeatTimestamp int64
	mutexWrite         sync.Mutex
	mutexRead          sync.Mutex
}
