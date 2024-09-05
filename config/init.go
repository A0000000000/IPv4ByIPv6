package config

import (
	"IPv4ByIPv6/socket"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

func GenConfigContext() (error, *ConfigContext) {
	return genConfigContext(true, "")
}

func GenConfigContextWithConfig(config string) (error, *ConfigContext) {
	return genConfigContext(false, config)
}

func genConfigContext(isMaster bool, config string) (error, *ConfigContext) {
	if isMaster {
		ctx := &ConfigContext{isMaster: true, config: Config{items: make(map[uint32]Item)}, syncSocketChannel: make(map[uint32]chan string)}
		err := ctx.initMasterContext()
		if err != nil {
			return err, nil
		}
		err = ctx.initExportsServer()
		if err != nil {
			return err, nil
		}
		err = ctx.checkConfigEnable()
		if err != nil {
			return err, nil
		}
		return nil, ctx
	} else {
		ctx := &ConfigContext{isMaster: false, config: Config{items: make(map[uint32]Item)}, syncSocketChannel: make(map[uint32]chan string)}
		decode, err := base64.StdEncoding.DecodeString(config)
		if err != nil {
			return err, nil
		}
		fmt.Sscanf(string(decode), "%s %d %d", &ctx.config.masterAddress, &ctx.config.masterPort, &ctx.config.segment)
		err = ctx.holdMasterConnection()
		if err != nil {
			return err, nil
		}
		err = ctx.requireNewNumber()
		return err, ctx
	}
}

func (ctx *ConfigContext) GetJoinConfig() string {
	config := fmt.Sprintf("%s %d %d", ctx.config.masterAddress, ctx.config.masterPort, ctx.config.segment)
	return base64.StdEncoding.EncodeToString([]byte(config))
}

func (ctx *ConfigContext) checkConfigEnable() error {
	if !ctx.isMaster || !ctx.socketCtx.isServerCtx {
		return errors.New("must master can use this interface.")
	}
	go func() {
		for {
			items := ctx.config.items
			newItems := make(map[uint32]Item)
			for k, v := range items {
				if v.socketCtx.isServerCtx || v.socketCtx.clientCtx.IsConnected() {
					newItems[k] = v
				}
			}
			ctx.config.items = newItems
			time.Sleep(5 * time.Second)
		}
	}()
	return nil
}

func (ctx *ConfigContext) initMasterContext() error {
	err, ipv6 := GetGlobalIPv6Address()
	if err != nil {
		return err
	}
	if ipv6 == nil {
		return errors.New("no IPv6 address")
	}
	ctx.config.masterAddress = ipv6.String()
	ctx.config.masterPort = ConfigPort
	ctx.config.segment = rand.Uint32()%155 + 100
	ctx.config.number = 1
	ctx.config.items[1] = Item{
		number:  ctx.config.number,
		address: ctx.config.masterAddress,
		port:    ListenPort,
	}
	return nil
}

func (ctx *ConfigContext) genNewNumber() uint32 {
	for {
		number := rand.Uint32() % 254
		_, has := ctx.config.items[number]
		if !has {
			return number
		}
	}
}

func (ctx *ConfigContext) initExportsServer() error {
	err, serverCtx := socket.CreateServer(true, ConfigPort, func(clientCtx *socket.ClientContext, data []byte) {
		go ctx.processClientData(clientCtx, string(data))
	})
	ctx.socketCtx = SocketContextHolder{serverCtx, nil, true}
	item, has := ctx.config.items[ctx.config.number]
	if has {
		item.socketCtx = ctx.socketCtx
		ctx.config.items[ctx.config.number] = item
	}
	if err != nil {
		return err
	}
	return nil
}

func (ctx *ConfigContext) processClientData(clientCtx *socket.ClientContext, data string) error {
	if len(data) <= 0 {
		return errors.New("empty data, ignore.")
	}
	cmds := strings.Split(data, " ")
	if len(cmds) == 0 {
		err := errors.New("empty params")
		log.Println("rev data error.", err)
		return err
	}
	switch cmds[0] {
	case CmdRequire:
		if !ctx.isMaster {
			clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %s", CmdRequireResult, CmdFailed)))
			return errors.New(fmt.Sprintf("%s not support no master.", CmdRequire))
		}
		if len(cmds) < 4 {
			clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %s %s", CmdRequireResult, CmdFailed, cmds[1])))
			return errors.New(fmt.Sprintf("%s params error.", CmdRequire))
		}
		ipv6 := cmds[2]
		var port uint32 = 0
		fmt.Sscanf(cmds[3], "%d", &port)
		number := ctx.genNewNumber()
		ctx.config.items[number] = Item{
			number:    number,
			address:   ipv6,
			port:      port,
			socketCtx: SocketContextHolder{serverCtx: nil, clientCtx: clientCtx, isServerCtx: false},
		}
		clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %s %s %s %d %d", CmdRequireResult, CmdSuccess, cmds[1], ipv6, number, port)))
		break
	case CmdRequireResult:
		if ctx.isMaster {
			return errors.New(fmt.Sprintf("%s not support master.", CmdRequireResult))
		}
		if len(cmds) < 3 {
			return errors.New(fmt.Sprintf("%s params count error.", CmdRequireResult))
		}
		var token uint32 = 0
		fmt.Sscanf(cmds[2], "%d", &token)
		ch, has := ctx.syncSocketChannel[token]
		if cmds[1] == CmdFailed {
			if has {
				ch <- data
			}
			return ctx.requireNewNumber()
		}
		if len(cmds) < 6 {
			if has {
				ch <- data
			}
			return ctx.requireNewNumber()
		}
		fmt.Sscanf(cmds[4], "%d", &ctx.config.number)
		if has {
			ch <- data
		}
		break
	case CmdQuery:
		if len(cmds) < 3 {
			return errors.New(fmt.Sprintf("%s params count error.", CmdQuery))
		}
		var number uint32 = 0
		fmt.Sscanf(cmds[2], "%d", &number)
		item, has := ctx.config.items[number]
		if has {
			clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %s %s %s %d %d", CmdQueryResult, CmdSuccess, cmds[1], item.address, item.number, item.port)))
		} else {
			clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %s %s", CmdQueryResult, CmdFailed, cmds[1])))
		}
		break
	case CmdQueryResult:
		var token uint32 = 0
		fmt.Sscanf(cmds[2], "%d", &token)
		ch, has := ctx.syncSocketChannel[token]
		if has {
			ch <- data
		}
		break
	default:
		err := errors.New("unknown cmd")
		log.Println("unknown cmd. rawdata = ", data)
		return err
	}
	return nil
}

func (ctx *ConfigContext) holdMasterConnection() error {
	err, clientCtx := socket.ConnectServer(true, ctx.config.masterAddress, ctx.config.masterPort, func(clientCtx *socket.ClientContext, data []byte) {
		go ctx.processClientData(clientCtx, string(data))
	})
	if err != nil {
		return err
	}
	ctx.socketCtx = SocketContextHolder{nil, clientCtx, false}
	return nil
}

func (ctx *ConfigContext) requireNewNumber() error {
	if ctx.isMaster {
		return errors.New("master cannot require new number")
	}
	err, ipv6Raw := GetGlobalIPv6Address()
	if err != nil {
		return err
	}
	token := rand.Uint32()
	for {
		_, has := ctx.syncSocketChannel[token]
		if !has {
			break
		}
	}
	ctx.syncSocketChannel[token] = make(chan string, 1)
	ctx.socketCtx.clientCtx.SendDataToServerDefault([]byte(fmt.Sprintf("%s %d %s %d", CmdRequire, token, ipv6Raw.String(), ListenPort)))
	<-ctx.syncSocketChannel[token]
	delete(ctx.syncSocketChannel, token)
	return nil
}

func (ctx *ConfigContext) QueryIPv6Address(number uint32) (error, Item, bool) {
	if ctx.isMaster {
		value, has := ctx.config.items[number]
		return nil, value, has
	}
	token := rand.Uint32()
	for {
		_, has := ctx.syncSocketChannel[token]
		if !has {
			break
		}
	}
	ctx.syncSocketChannel[token] = make(chan string, 1)
	sendData := fmt.Sprintf("%s %d %d", CmdQuery, token, number)
	ctx.socketCtx.clientCtx.SendDataToServerDefault([]byte(sendData))
	result := <-ctx.syncSocketChannel[token]
	delete(ctx.syncSocketChannel, token)
	cmd := strings.Split(result, " ")
	if len(cmd) < 6 {
		return errors.New("cmd params count error"), Item{}, false
	}
	if cmd[0] == CmdQueryResult && cmd[1] == CmdSuccess {
		res := Item{}
		res.address = cmd[3]
		fmt.Sscanf(cmd[4], "%d", &res.number)
		fmt.Sscanf(cmd[5], "%d", &res.port)
		return nil, res, res.number == number
	}
	return nil, Item{}, false
}

func (ctx *ConfigContext) GetSegment() uint32 {
	return ctx.config.segment
}

func (ctx *ConfigContext) GetNumber() uint32 {
	return ctx.config.number
}

func (ctx Item) GetIPv6Address() string {
	return ctx.address
}

func (ctx Item) GetIPv6Port() uint32 {
	return ctx.port
}
