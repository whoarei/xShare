// discovery 包的单元测试和集成测试。
// 单元测试覆盖 Peer/IPInfo JSON 序列化、字段标签正确性、ServiceName 常量。
// 集成测试通过 Register + Discover 往返验证 mDNS 服务注册和自发现能力。
package discovery

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestPeerJSON 验证 Peer 结构体的 JSON 序列化/反序列化往返正确性。
// 覆盖全字段 peer、空 peer、零端口 peer 三种场景。
func TestPeerJSON(t *testing.T) {
	tests := []struct {
		name string
		peer Peer
	}{
		{
			name: "full peer",
			peer: Peer{
				Name: "my-hostname",
				Host: "192.168.1.100",
				Port: 9527,
				Addr: "192.168.1.100:9527",
			},
		},
		{
			name: "empty peer",
			peer: Peer{},
		},
		{
			name: "zero port",
			peer: Peer{
				Name: "test",
				Host: "10.0.0.1",
				Port: 0,
				Addr: "10.0.0.1:0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.peer)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded Peer
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.Name != tt.peer.Name {
				t.Errorf("Name mismatch: got %q, want %q", decoded.Name, tt.peer.Name)
			}
			if decoded.Host != tt.peer.Host {
				t.Errorf("Host mismatch: got %q, want %q", decoded.Host, tt.peer.Host)
			}
			if decoded.Port != tt.peer.Port {
				t.Errorf("Port mismatch: got %d, want %d", decoded.Port, tt.peer.Port)
			}
			if decoded.Addr != tt.peer.Addr {
				t.Errorf("Addr mismatch: got %q, want %q", decoded.Addr, tt.peer.Addr)
			}
		})
	}
}

// TestPeerJSON_FieldTags 验证 Peer 结构体的 JSON 字段标签输出正确的 key 名称。
// 确保 name、host、port、addr 四个字段以小写 JSON key 序列化。
func TestPeerJSON_FieldTags(t *testing.T) {
	p := Peer{
		Name: "peer1",
		Host: "192.168.0.1",
		Port: 8080,
		Addr: "192.168.0.1:8080",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	expectedKeys := []string{"name", "host", "port", "addr"}
	for _, k := range expectedKeys {
		if _, ok := m[k]; !ok {
			t.Errorf("missing JSON key %q in output: %s", k, data)
		}
	}

	if v, ok := m["port"].(float64); ok && int(v) != p.Port {
		t.Errorf("port value: got %v, want %d", v, p.Port)
	}
}

// TestServiceName 验证 mDNS 服务名常量格式正确。
// 必须为 "_xshare._tcp" 且遵循 _service._proto 命名规范。
func TestServiceName(t *testing.T) {
	if ServiceName != "_xshare._tcp" {
		t.Errorf("ServiceName = %q, want %q", ServiceName, "_xshare._tcp")
	}

	// mDNS service names must follow the format _service._proto
	if len(ServiceName) < 6 {
		t.Errorf("ServiceName too short: %q", ServiceName)
	}
}

// TestIPInfoJSON 验证 IPInfo 结构体的 JSON 序列化/反序列化往返正确性。
// 覆盖 IPv4、IPv6 两种场景。
func TestIPInfoJSON(t *testing.T) {
	tests := []struct {
		name string
		info IPInfo
	}{
		{
			name: "ipv4",
			info: IPInfo{
				IP:     "192.168.1.100",
				Iface:  "eth0",
				Family: "v4",
			},
		},
		{
			name: "ipv6",
			info: IPInfo{
				IP:     "2001:db8::1",
				Iface:  "eth0",
				Family: "v6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded IPInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if decoded.IP != tt.info.IP {
				t.Errorf("IP mismatch: got %q, want %q", decoded.IP, tt.info.IP)
			}
			if decoded.Iface != tt.info.Iface {
				t.Errorf("Iface mismatch: got %q, want %q", decoded.Iface, tt.info.Iface)
			}
			if decoded.Family != tt.info.Family {
				t.Errorf("Family mismatch: got %q, want %q", decoded.Family, tt.info.Family)
			}
		})
	}
}

// TestIPInfoJSON_FieldTags 验证 IPInfo 的 JSON 字段标签输出正确的 key 名称。
// 确保 ip、iface、family 三个字段以小写 JSON key 序列化。
func TestIPInfoJSON_FieldTags(t *testing.T) {
	info := IPInfo{
		IP:     "10.0.0.1",
		Iface:  "wlan0",
		Family: "v4",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	expectedKeys := []string{"ip", "iface", "family"}
	for _, k := range expectedKeys {
		if _, ok := m[k]; !ok {
			t.Errorf("missing JSON key %q in output: %s", k, data)
		}
	}

	if v, ok := m["ip"].(string); !ok || v != "10.0.0.1" {
		t.Errorf("ip value: got %v, want %q", v, "10.0.0.1")
	}
	if v, ok := m["family"].(string); !ok || v != "v4" {
		t.Errorf("family value: got %v, want %q", v, "v4")
	}
}

// TestListIPs 验证 ListIPs 返回非空结果，且过滤 loopback/link-local/multicast/unspecified。
func TestListIPs(t *testing.T) {
	ips, err := ListIPs()
	if err != nil {
		t.Fatalf("ListIPs failed: %v", err)
	}

	for _, info := range ips {
		if info.IP == "" {
			t.Error("IP must not be empty")
		}
		if info.Iface == "" {
			t.Error("Iface must not be empty")
		}

		parsed := net.ParseIP(info.IP)
		if parsed == nil {
			t.Errorf("Invalid IP string: %q", info.IP)
			continue
		}
		if parsed.IsLoopback() {
			t.Errorf("loopback IP %q must not appear", info.IP)
		}
		if parsed.IsLinkLocalUnicast() {
			t.Errorf("link-local IP %q must not appear", info.IP)
		}
		if parsed.IsMulticast() {
			t.Errorf("multicast IP %q must not appear", info.IP)
		}
		if parsed.IsUnspecified() {
			t.Errorf("unspecified IP %q must not appear", info.IP)
		}
		if info.Family != "v4" && info.Family != "v6" {
			t.Errorf("Family must be v4 or v6: got %q for IP %q", info.Family, info.IP)
		}
	}
	t.Logf("Found %d IP address(es)", len(ips))
}

// TestListIPs_NoLoopback 验证 ListIPs 不返回 loopback 地址。
func TestListIPs_NoLoopback(t *testing.T) {
	ips, err := ListIPs()
	if err != nil {
		t.Fatalf("ListIPs failed: %v", err)
	}

	for _, info := range ips {
		parsed := net.ParseIP(info.IP)
		if parsed != nil && parsed.IsLoopback() {
			t.Errorf("loopback address %q must not appear", info.IP)
		}
	}
}

// TestResolveIP 验证 resolveIP 能正确查找真实网卡上的 IP。
func TestResolveIP(t *testing.T) {
	ips, err := ListIPs()
	if err != nil {
		t.Fatalf("ListIPs failed: %v", err)
	}
	if len(ips) == 0 {
		t.Skip("no IPs available for resolveIP test")
	}

	info := ips[0]
	parsedIP, iface, err := resolveIP(info.IP)
	if err != nil {
		t.Fatalf("resolveIP(%q) failed: %v", info.IP, err)
	}
	if parsedIP == nil {
		t.Fatal("resolveIP returned nil IP")
	}
	if !parsedIP.Equal(net.ParseIP(info.IP)) {
		t.Errorf("resolveIP returned wrong IP: got %v, want %v", parsedIP, info.IP)
	}
	if iface == nil {
		t.Fatal("resolveIP returned nil interface")
	}
	if iface.Name != info.Iface {
		t.Errorf("resolveIP returned wrong interface: got %q, want %q", iface.Name, info.Iface)
	}
}

// TestResolveIP_Nonexistent 验证 resolveIP 对不存在的 IP 返回错误。
func TestResolveIP_Nonexistent(t *testing.T) {
	_, _, err := resolveIP("203.0.113.99")
	if err == nil {
		t.Error("expected error for nonexistent IP, got nil")
	}
}

// TestResolveIP_InvalidFormat 验证 resolveIP 对无效格式返回错误。
func TestResolveIP_InvalidFormat(t *testing.T) {
	_, _, err := resolveIP("not-an-ip")
	if err == nil {
		t.Error("expected error for invalid IP format, got nil")
	}
}

// TestRegisterAndDiscover 集成测试：注册 mDNS 服务后通过 Discover 自发现。
// 验证 Register → Discover 往返流程正确，发现的 peer 字段非空。
// 需要 mDNS 网络支持（局域网组播可达）。
func TestRegisterAndDiscover(t *testing.T) {
	const testPort = 19528

	server, err := Register(testPort, "")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer server.Shutdown()

	// Wait for mDNS server to fully start broadcasting
	time.Sleep(500 * time.Millisecond)

	timeout := 4 * time.Second
	peers, err := Discover(timeout, "")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	t.Logf("Discovered %d peers", len(peers))

	found := false
	for _, p := range peers {
		t.Logf("  peer: name=%s host=%s port=%d addr=%s", p.Name, p.Host, p.Port, p.Addr)
		if p.Port == testPort {
			found = true
			if p.Addr == "" {
				t.Errorf("Peer Addr is empty for port %d", testPort)
			}
			if p.Host == "" {
				t.Errorf("Peer Host is empty for port %d", testPort)
			}
			if p.Name == "" {
				t.Errorf("Peer Name is empty for port %d", testPort)
			}
			break
		}
	}
	if !found {
		t.Error("Could not find self among discovered peers")
	}
}

// TestRegisterWithSpecificIP 集成测试：绑定指定 IP 注册 mDNS，验证广告 IP 仅为此 IP。
func TestRegisterWithSpecificIP(t *testing.T) {
	ips, err := ListIPs()
	if err != nil {
		t.Fatalf("ListIPs failed: %v", err)
	}
	if len(ips) == 0 {
		t.Skip("no IPs available for specific-IP test")
	}

	// Pick first IPv4 address
	var testIP string
	for _, info := range ips {
		if info.Family == "v4" {
			testIP = info.IP
			break
		}
	}
	if testIP == "" {
		t.Skip("no IPv4 address available")
	}

	const testPort = 19529

	server, err := Register(testPort, testIP)
	if err != nil {
		t.Fatalf("Register with IP %q failed: %v", testIP, err)
	}
	defer server.Shutdown()

	time.Sleep(500 * time.Millisecond)

	timeout := 4 * time.Second
	peers, err := Discover(timeout, "")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	t.Logf("Discovered %d peers (looking for port %d on IP %s)", len(peers), testPort, testIP)

	found := false
	for _, p := range peers {
		t.Logf("  peer: name=%s host=%s port=%d addr=%s", p.Name, p.Host, p.Port, p.Addr)
		if p.Port == testPort {
			found = true
			if p.Host != testIP {
				t.Errorf("Peer Host should be %q, got %q", testIP, p.Host)
			}
			if p.Addr == "" {
				t.Errorf("Peer Addr is empty for port %d", testPort)
			}
			if p.Name == "" {
				t.Errorf("Peer Name is empty for port %d", testPort)
			}
			break
		}
	}
	if !found {
		t.Errorf("Could not find self (port %d, IP %s) among discovered peers", testPort, testIP)
	}
}
