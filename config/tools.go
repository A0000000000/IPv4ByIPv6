package config

import (
	"net"
)

func GetGlobalIPv6Address() (error, net.IP) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err, nil
	}
	for _, ifce := range interfaces {
		addrs, err := ifce.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ok := addr.(*net.IPNet)
			if !ok || ip.IP.IsLoopback() {
				continue
			}
			if ip.IP.To4() != nil {
				continue
			}
			ones, _ := ip.Mask.Size()
			if ip.IP.IsGlobalUnicast() && ones != 128 {
				return nil, ip.IP
			}
		}
	}
	return nil, nil
}
