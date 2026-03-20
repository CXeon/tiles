package ip

import (
	"errors"
	"net"
)

// GetLocalIP returns the first non-loopback, non-docker-bridge IPv4 address
// by enumerating network interfaces. Works on Linux, macOS, and Windows.
func GetLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			// Skip Docker bridge IPs (172.17.0.0/16)
			if ip[0] == 172 && ip[1] == 17 {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("no valid local IP found")
}
