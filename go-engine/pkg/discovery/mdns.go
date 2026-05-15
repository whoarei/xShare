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

func ListIPs() ([]IPInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var out []IPInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip.IsLoopback() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
				continue
			}

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

func Register(port int, bindIP string) (*mdns.Server, error) {
	host, _ := os.Hostname()
	info := []string{"xShare v1"}

	var ips []net.IP
	var iface *net.Interface

	if bindIP != "" {
		parsedIP, foundIface, err := resolveIP(bindIP)
		if err != nil {
			return nil, fmt.Errorf("resolve IP: %w", err)
		}
		ips = []net.IP{parsedIP}
		iface = foundIface
	}

	service, err := mdns.NewMDNSService(host, ServiceName, "", "", port, ips, info)
	if err != nil {
		return nil, fmt.Errorf("create mdns service: %w", err)
	}
	server, err := mdns.NewServer(&mdns.Config{Zone: service, Iface: iface})
	if err != nil {
		return nil, fmt.Errorf("start mdns server: %w", err)
	}
	return server, nil
}

func Discover(timeout time.Duration, bindIP string) ([]Peer, error) {
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

	var iface *net.Interface
	if bindIP != "" {
		var err error
		_, iface, err = resolveIP(bindIP)
		if err != nil {
			close(entriesCh)
			<-done
			return nil, fmt.Errorf("resolve IP: %w", err)
		}
	}

	params := &mdns.QueryParam{
		Service:             ServiceName,
		Timeout:             timeout,
		Entries:             entriesCh,
		WantUnicastResponse: true,
		Interface:           iface,
	}

	if err := mdns.Query(params); err != nil {
		close(entriesCh)
		<-done
		return nil, fmt.Errorf("mdns query: %w", err)
	}

	func() {
		defer func() { _ = recover() }()
		close(entriesCh)
	}()
	<-done
	return peers, nil
}
