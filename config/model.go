package config

import "IPv4ByIPv6/socket"

const ConfigPort uint32 = 925
const ListenPort uint32 = 1113
const CmdRequire = "require"
const CmdRequireResult = "require_result"
const CmdQuery = "query"
const CmdQueryResult = "query_result"
const CmdSuccess = "success"
const CmdFailed = "failed"

type ConfigContext struct {
	isMaster          bool   // 是否是master节点
	config            Config // 具体配置
	socketCtx         SocketContextHolder
	syncSocketChannel map[uint32]chan string
}

type Config struct {
	segment       uint32          // ip段
	number        uint32          // ip号
	masterAddress string          // master节点的ipv6地址
	masterPort    uint32          // master节点操作配置的端口
	items         map[uint32]Item // master节点维护的所有配置
}

type Item struct {
	number    uint32 // ip号
	address   string // ipv6地址
	port      uint32 // 通信转发端口
	socketCtx SocketContextHolder
}

type SocketContextHolder struct {
	serverCtx   *socket.ServerContext
	clientCtx   *socket.ClientContext
	isServerCtx bool
}

type StringChannel struct {
	channel chan string
}
