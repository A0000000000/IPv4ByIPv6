package dispatch

import (
	"IPv4ByIPv6/config"
	"IPv4ByIPv6/eth"
	"IPv4ByIPv6/socket"
	"sync"
)

type DispatchContext struct {
	configCtx      *config.ConfigContext
	ethCtx         *eth.EthContext
	socketCtx      *socket.ServerContext
	mutexWrite     sync.Mutex
	mutexRead      sync.Mutex
	clientCtxCache map[uint32]*socket.ClientContext
}
