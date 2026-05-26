package discovery

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
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

type ServerGroup struct {
	servers []*mdns.Server
}

func (sg *ServerGroup) Shutdown() {
	for _, s := range sg.servers {
		s.Shutdown()
	}
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

type ifaceConfig struct {
	ips   []net.IP
	iface *net.Interface
}

func collectIfaceConfigs(bindIP string) ([]ifaceConfig, error) {
	if bindIP != "" {
		parsedIP, foundIface, err := resolveIP(bindIP)
		if err != nil {
			return nil, fmt.Errorf("resolve IP: %w", err)
		}
		return []ifaceConfig{{ips: []net.IP{parsedIP}, iface: foundIface}}, nil
	}

	ifaces, err := listLANInterfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no suitable network interface found for mDNS")
	}

	var configs []ifaceConfig
	for _, iface := range ifaces {
		ips := interfaceIPs(&iface)
		if len(ips) == 0 {
			continue
		}
		configs = append(configs, ifaceConfig{ips: ips, iface: &iface})
	}
	return configs, nil
}

func Register(port int, bindIP string) (*ServerGroup, error) {
	host, _ := os.Hostname()
	info := []string{"xShare v1"}

	configs, err := collectIfaceConfigs(bindIP)
	if err != nil {
		return nil, err
	}

	var servers []*mdns.Server
	for _, cfg := range configs {
		service, err := mdns.NewMDNSService(
			host, ServiceName, "", "", port, cfg.ips, info,
		)
		if err != nil {
			continue
		}
		server, err := mdns.NewServer(&mdns.Config{Zone: service, Iface: cfg.iface})
		if err != nil {
			continue
		}
		servers = append(servers, server)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("failed to start mdns server on any interface")
	}

	return &ServerGroup{servers: servers}, nil
}

func Discover(timeout time.Duration, bindIP string) ([]Peer, error) {
	var ifaces []net.Interface

	if bindIP != "" {
		_, iface, err := resolveIP(bindIP)
		if err != nil {
			return nil, fmt.Errorf("resolve IP: %w", err)
		}
		ifaces = []net.Interface{*iface}
	} else {
		lanIfaces, err := listLANInterfaces()
		if err != nil {
			return nil, fmt.Errorf("list interfaces: %w", err)
		}
		ifaces = lanIfaces
	}

	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no suitable network interface found for mDNS")
	}

	entriesCh := make(chan *mdns.ServiceEntry, 32)
	var peers []Peer
	var mu sync.Mutex

	done := make(chan struct{})
	go func() {
		for entry := range entriesCh {
			addr := entry.AddrV4
			if addr == nil {
				addr = entry.AddrV6
			}
			if addr != nil {
				mu.Lock()
				peers = append(peers, Peer{
					Name: entry.Host,
					Host: addr.String(),
					Port: entry.Port,
					Addr: fmt.Sprintf("%s:%d", addr, entry.Port),
				})
				mu.Unlock()
			}
		}
		close(done)
	}()

	perIfaceTimeout := timeout / time.Duration(len(ifaces))
	if perIfaceTimeout < 1*time.Second {
		perIfaceTimeout = 1 * time.Second
	}

	for _, iface := range ifaces {
		params := &mdns.QueryParam{
			Service:             ServiceName,
			Timeout:             perIfaceTimeout,
			Entries:             entriesCh,
			WantUnicastResponse: true,
			Interface:           &iface,
		}
		mdns.Query(params)
	}

	func() {
		defer func() { _ = recover() }()
		close(entriesCh)
	}()
	<-done

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
