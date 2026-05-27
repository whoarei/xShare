package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const ServiceName = "_xshare._tcp"

type Peer struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
	Addr string `json:"addr"`
}

type IPInfo struct {
	IP     string `json:"ip"`
	Iface  string `json:"iface"`
	Family string `json:"family"`
}

type InterfaceInfo struct {
	Name  string   `json:"name"`
	Index int      `json:"index"`
	MTU   int      `json:"mtu"`
	Flags string   `json:"flags"`
	MAC   string   `json:"mac"`
	IPs   []IPInfo `json:"ips"`
}

func isValidUnicastIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsMulticast() &&
		!ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

func listLANInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var out []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if len(interfaceIPs(&iface)) == 0 {
			continue
		}
		out = append(out, iface)
	}
	return out, nil
}

func interfaceIPs(iface *net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	var ips []net.IP
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP
		if isValidUnicastIP(ip) {
			ips = append(ips, ip)
		}
	}
	return ips
}

func ListIPs() ([]IPInfo, error) {
	ifaces, err := listLANInterfaces()
	if err != nil {
		return nil, err
	}

	var out []IPInfo
	for _, iface := range ifaces {
		for _, ip := range interfaceIPs(&iface) {
			family := "v6"
			if ip.To4() != nil {
				family = "v4"
			}
			out = append(out, IPInfo{
				IP:     ip.String(),
				Iface:  iface.Name,
				Family: family,
			})
		}
	}
	return out, nil
}

func flagToString(flags net.Flags) string {
	names := []string{}
	if flags&net.FlagUp != 0 {
		names = append(names, "up")
	}
	if flags&net.FlagBroadcast != 0 {
		names = append(names, "broadcast")
	}
	if flags&net.FlagLoopback != 0 {
		names = append(names, "loopback")
	}
	if flags&net.FlagPointToPoint != 0 {
		names = append(names, "pointtopoint")
	}
	if flags&net.FlagMulticast != 0 {
		names = append(names, "multicast")
	}
	if flags&net.FlagRunning != 0 {
		names = append(names, "running")
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, "|")
}

func ListInterfaces() ([]InterfaceInfo, error) {
	all, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var out []InterfaceInfo
	for _, iface := range all {
		var ips []IPInfo
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				ip := ipnet.IP
				family := "v6"
				if ip.To4() != nil {
					family = "v4"
				}
				ips = append(ips, IPInfo{
					IP:     ip.String(),
					Iface:  iface.Name,
					Family: family,
				})
			}
		}

		mac := ""
		if iface.HardwareAddr != nil {
			mac = iface.HardwareAddr.String()
		}

		out = append(out, InterfaceInfo{
			Name:  iface.Name,
			Index: iface.Index,
			MTU:   iface.MTU,
			Flags: flagToString(iface.Flags),
			MAC:   mac,
			IPs:   ips,
		})
	}
	return out, nil
}

func resolveIP(ipStr string) (net.IP, *net.Interface, error) {
	targetIP := net.ParseIP(ipStr)
	if targetIP == nil {
		return nil, nil, fmt.Errorf("invalid IP: %q", ipStr)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, fmt.Errorf("list interfaces: %w", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && ipnet.IP.Equal(targetIP) {
				return targetIP, &iface, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("IP %q not found on any interface", ipStr)
}

func collectInterfaces(bindIP string) ([]net.Interface, error) {
	if bindIP != "" {
		_, iface, err := resolveIP(bindIP)
		if err != nil {
			return nil, fmt.Errorf("resolve IP: %w", err)
		}
		return []net.Interface{*iface}, nil
	}

	ifaces, err := listLANInterfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no suitable network interface found for mDNS")
	}
	return ifaces, nil
}

func collectIPs(bindIP string) []string {
	if bindIP != "" {
		return []string{bindIP}
	}
	ips, err := ListIPs()
	if err != nil {
		return nil
	}
	var out []string
	for _, info := range ips {
		out = append(out, info.IP)
	}
	return out
}

func Register(port int, bindIP string) (*zeroconf.Server, error) {
	host, _ := os.Hostname()

	ifaces, err := collectInterfaces(bindIP)
	if err != nil {
		return nil, err
	}

	ips := collectIPs(bindIP)

	server, err := zeroconf.RegisterProxy(
		host,
		ServiceName,
		"local.",
		port,
		host+".",
		ips,
		[]string{"xShare v1"},
		ifaces,
	)
	if err != nil {
		return nil, fmt.Errorf("zeroconf register: %w", err)
	}

	return server, nil
}

func Discover(timeout time.Duration, bindIP string) ([]Peer, error) {
	ifaces, err := collectInterfaces(bindIP)
	if err != nil {
		return nil, err
	}

	resolver, err := zeroconf.NewResolver(
		zeroconf.SelectIfaces(ifaces),
	)
	if err != nil {
		return nil, fmt.Errorf("zeroconf resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	var peers []Peer

	go func() {
		for entry := range entries {
			var addr net.IP
			if len(entry.AddrIPv4) > 0 {
				addr = entry.AddrIPv4[0]
			} else if len(entry.AddrIPv6) > 0 {
				addr = entry.AddrIPv6[0]
			}
			if addr != nil {
				peers = append(peers, Peer{
					Name: entry.HostName,
					Host: addr.String(),
					Port: entry.Port,
					Addr: fmt.Sprintf("%s:%d", addr, entry.Port),
				})
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = resolver.Browse(ctx, ServiceName, "local.", entries)
	if err != nil {
		return nil, fmt.Errorf("zeroconf browse: %w", err)
	}
	<-ctx.Done()

	seen := make(map[string]bool)
	var result []Peer
	for _, p := range peers {
		if !seen[p.Addr] {
			seen[p.Addr] = true
			result = append(result, p)
		}
	}

	return result, nil
}
