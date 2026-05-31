// mDNS设备发现模块
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

const ServiceName = "_xshare._tcp" // mDNS服务名称

// Peer 发现的对端设备信息
type Peer struct {
	Name string `json:"name"` // 主机名
	Host string `json:"host"` // IP地址
	Port int    `json:"port"` // 端口号
	Addr string `json:"addr"` // 完整地址 host:port
}

// IPInfo IP地址信息
type IPInfo struct {
	IP     string `json:"ip"`     // IP地址
	Iface  string `json:"iface"`  // 网络接口名称
	Family string `json:"family"` // 地址族 v4/v6
}

// InterfaceInfo 网络接口详细信息
type InterfaceInfo struct {
	Name  string   `json:"name"`  // 接口名称
	Index int      `json:"index"` // 接口索引
	MTU   int      `json:"mtu"`   // 最大传输单元
	Flags string   `json:"flags"` // 接口标志
	MAC   string   `json:"mac"`   // MAC地址
	IPs   []IPInfo `json:"ips"`   // IP地址列表
}

// isValidUnicastIP 检查是否为有效的单播IP地址
func isValidUnicastIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsMulticast() &&
		!ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

// listLANInterfaces 列出可用的局域网网络接口
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

// interfaceIPs 获取网络接口的所有有效IP地址
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

// ListIPs 列出本机所有可用的IP地址
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

// flagToString 将网络接口标志转换为可读字符串
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

// ListInterfaces 列出本机所有网络接口详细信息
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

// resolveIP 解析IP地址并找到对应的网络接口
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

// collectInterfaces 收集要使用的网络接口
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

// collectIPs 收集要使用的IP地址
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

// Register 注册mDNS服务，使其他设备可以发现本机
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

// Discover 发现局域网内的其他xShare设备
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
			for _, ip := range entry.AddrIPv4 {
				if !isValidUnicastIP(ip) {
					continue
				}
				peers = append(peers, Peer{
					Name: entry.HostName,
					Host: ip.String(),
					Port: entry.Port,
					Addr: fmt.Sprintf("%s:%d", ip, entry.Port),
				})
			}
			for _, ip := range entry.AddrIPv6 {
				if !isValidUnicastIP(ip) {
					continue
				}
				peers = append(peers, Peer{
					Name: entry.HostName,
					Host: ip.String(),
					Port: entry.Port,
					Addr: fmt.Sprintf("%s:%d", ip, entry.Port),
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

	// 去重处理
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
