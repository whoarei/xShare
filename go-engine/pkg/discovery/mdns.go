package discovery

import (
	"fmt"
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

func Register(port int) (*mdns.Server, error) {
	host, _ := os.Hostname()
	info := []string{"xShare v1"}
	service, err := mdns.NewMDNSService(host, ServiceName, "", "", port, nil, info)
	if err != nil {
		return nil, fmt.Errorf("create mdns service: %w", err)
	}
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("start mdns server: %w", err)
	}
	return server, nil
}

func Discover(timeout time.Duration) ([]Peer, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 32)
	var peers []Peer
	var mu sync.Mutex

	done := make(chan struct{})
	go func() {
		for entry := range entriesCh {
			if len(entry.AddrV4) > 0 {
				mu.Lock()
				peers = append(peers, Peer{
					Name: entry.Host,
					Host: entry.AddrV4.String(),
					Port: entry.Port,
					Addr: fmt.Sprintf("%s:%d", entry.AddrV4, entry.Port),
				})
				mu.Unlock()
			}
		}
		close(done)
	}()

	params := &mdns.QueryParam{
		Service:             ServiceName,
		Timeout:             timeout,
		Entries:             entriesCh,
		WantUnicastResponse: true,
	}

	if err := mdns.Query(params); err != nil {
		close(entriesCh)
		<-done
		return nil, fmt.Errorf("mdns query: %w", err)
	}

	<-done
	return peers, nil
}
