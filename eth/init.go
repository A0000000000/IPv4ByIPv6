package eth

import (
	"IPv4ByIPv6/eth/platform"
)

func CreateTunDevice(segment, number uint32) (error, *EthContext) {
	const TunName = "IPv4ByIPv6"
	// 准备数据
	err, rwc := platform.CreateTunDeviceInner(TunName, segment, number)
	if err != nil {
		return err, nil
	}
	return nil, &EthContext{
		TunName,
		rwc,
	}
}
