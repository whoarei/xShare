// main包的单元测试
// 测试命令行参数解析和CLI命令输出
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-engine/pkg/discovery"
)

// TestGetArg 测试命令行参数提取函数
// 验证各种参数格式的正确解析，包括精确匹配、前缀匹配、默认值等
func TestGetArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		key      string
		defVal   string
		expected string
	}{
		{
			// 测试场景：精确匹配参数名
			// 验证点：从多个参数中正确提取目标参数值
			name:     "exact match",
			args:     []string{"--port=9527", "--dir=./received"},
			key:      "port",
			defVal:   "",
			expected: "9527",
		},
		{
			// 测试场景：参数名是另一个参数的前缀
			// 验证点：不会误匹配以目标key开头的其他参数(如--port-num不匹配--port)
			name:     "match with prefix",
			args:     []string{"--port-num=8080", "--port=9527"},
			key:      "port",
			defVal:   "",
			expected: "9527",
		},
		{
			// 测试场景：参数不存在
			// 验证点：当目标参数不在args中时，返回默认值
			name:     "not found returns default",
			args:     []string{"--port=9527"},
			key:      "dir",
			defVal:   "./received",
			expected: "./received",
		},
		{
			// 测试场景：空参数列表
			// 验证点：当args为空切片时，返回默认值
			name:     "empty args returns default",
			args:     []string{},
			key:      "timeout",
			defVal:   "5",
			expected: "5",
		},
		{
			// 测试场景：IPv4地址参数
			// 验证点：正确提取包含点号的IPv4地址值
			name:     "ip argument (v4)",
			args:     []string{"--ip=192.168.1.100", "--timeout=3"},
			key:      "ip",
			defVal:   "",
			expected: "192.168.1.100",
		},
		{
			// 测试场景：IPv6地址参数
			// 验证点：正确提取包含冒号的IPv6地址值
			name:     "ip argument (v6)",
			args:     []string{"--ip=2001:db8::1", "--port=8080"},
			key:      "ip",
			defVal:   "",
			expected: "2001:db8::1",
		},
		{
			// 测试场景：参数值为空字符串
			// 验证点：当参数格式为"--key="时，值为空串，应回退到默认值
			name:     "empty value falls back to default",
			args:     []string{"--port="},
			key:      "port",
			defVal:   "9527",
			expected: "9527",
		},
		{
			// 测试场景：目标key是其他参数名的前缀
			// 验证点：key="port"不会匹配"--port-range"，确保前缀精确匹配
			name:     "key is prefix of another arg",
			args:     []string{"--port-range=1000-2000"},
			key:      "port",
			defVal:   "",
			expected: "",
		},
		{
			// 测试场景：nil参数切片
			// 验证点：传入nil不会panic，应返回默认值
			name:     "nil args returns default",
			args:     nil,
			key:      "port",
			defVal:   "9527",
			expected: "9527",
		},
		{
			// 测试场景：存在多个匹配的参数
			// 验证点：返回第一个匹配项（遍历顺序）
			name:     "multiple matches returns first",
			args:     []string{"--port=8080", "--port=9527"},
			key:      "port",
			defVal:   "",
			expected: "8080",
		},
		{
			// 测试场景：参数值中包含等号
			// 验证点：从第一个=处切分，值中保留后续的等号
			name:     "value contains equals sign",
			args:     []string{"--key=a=b=c"},
			key:      "key",
			defVal:   "",
			expected: "a=b=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArg(tt.args, tt.key, tt.defVal)
			if result != tt.expected {
				t.Errorf("getArg(%v, %q, %q) = %q, want %q",
					tt.args, tt.key, tt.defVal, result, tt.expected)
			}
		})
	}
}

// TestCmdListIPs 测试列出IP地址命令
// 验证输出格式为有效的JSON，且包含必要的字段(ip, iface, family)
func TestCmdListIPs(t *testing.T) {
	// 捕获stdout输出用于验证
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmdListIPs()

	w.Close()
	os.Stdout = origStdout

	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}
	output := string(buf)

	// 验证点：输出不能为空，确保命令产生了结果
	if output == "" {
		t.Fatal("Expected non-empty stdout output")
	}

	var parsed map[string]interface{}
	// 验证点：输出必须是有效的JSON格式
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Stdout is not valid JSON: %v\nOutput: %s", err, output)
	}

	// 验证点：JSON的type字段必须为"ips"，标识这是IP列表响应
	if typ, ok := parsed["type"].(string); !ok || typ != "ips" {
		t.Errorf("Expected type=ips, got %v", parsed["type"])
	}

	ips, ok := parsed["ips"].([]interface{})
	// 验证点：必须存在ips字段且为数组类型
	if !ok {
		t.Fatal("ips field missing or not an array")
	}

	t.Logf("Listed %d IP(s)", len(ips))

	for _, raw := range ips {
		m, ok := raw.(map[string]interface{})
		// 验证点：每个IP条目必须是JSON对象
		if !ok {
			t.Errorf("IP entry is not a JSON object: %v", raw)
			continue
		}
		// 验证点：每个IP条目必须包含ip字段（IP地址）
		ipStr, ok := m["ip"].(string)
		if !ok || ipStr == "" {
			t.Error("IP entry missing or empty 'ip' field")
		} else if net.ParseIP(ipStr) == nil {
			t.Errorf("IP entry 'ip' is not a valid IP address: %q", ipStr)
		}
		// 验证点：每个IP条目必须包含iface字段（网络接口名）
		ifaceStr, ok := m["iface"].(string)
		if !ok || ifaceStr == "" {
			t.Error("IP entry missing or empty 'iface' field")
		}
		// 验证点：每个IP条目必须包含family字段且为v4或v6
		family, ok := m["family"].(string)
		if !ok {
			t.Error("IP entry missing 'family' field")
		} else if family != "v4" && family != "v6" {
			t.Errorf("IP entry 'family' must be 'v4' or 'v6', got %q", family)
		}
	}
}

// TestCmdListIfaces 测试列出网络接口命令
// 验证输出格式为有效的JSON，且包含必要的字段(name, index, mtu, flags, mac, ips)
func TestCmdListIfaces(t *testing.T) {
	// 捕获stdout输出用于验证
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmdListIfaces()

	w.Close()
	os.Stdout = origStdout

	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}
	output := string(buf)

	// 验证点：输出不能为空，确保命令产生了结果
	if output == "" {
		t.Fatal("Expected non-empty stdout output")
	}

	var parsed map[string]interface{}
	// 验证点：输出必须是有效的JSON格式
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Stdout is not valid JSON: %v\nOutput: %s", err, output)
	}

	// 验证点：JSON的type字段必须为"interfaces"，标识这是接口列表响应
	if typ, ok := parsed["type"].(string); !ok || typ != "interfaces" {
		t.Errorf("Expected type=interfaces, got %v", parsed["type"])
	}

	ifaces, ok := parsed["interfaces"].([]interface{})
	// 验证点：必须存在interfaces字段且为数组类型
	if !ok {
		t.Fatal("interfaces field missing or not an array")
	}

	t.Logf("Listed %d interface(s)", len(ifaces))

	// 验证点：本机至少应有一个网络接口
	if len(ifaces) == 0 {
		t.Fatal("expected at least one network interface")
	}

	for _, raw := range ifaces {
		m, ok := raw.(map[string]interface{})
		// 验证点：每个接口条目必须是JSON对象
		if !ok {
			t.Errorf("interface entry is not a JSON object: %v", raw)
			continue
		}
		// 验证点：必须包含name字段（接口名称，如eth0）
		if _, ok := m["name"].(string); !ok {
			t.Error("interface entry missing or non-string 'name' field")
		}
		// 验证点：必须包含index字段且为正整数
		index, ok := m["index"].(float64)
		if !ok {
			t.Error("interface entry missing or non-numeric 'index' field")
		} else if int(index) <= 0 {
			t.Errorf("interface index must be positive, got %d", int(index))
		}
		// 验证点：必须包含mtu字段且为整数（部分虚拟接口可能为-1表示未知）
		mtu, ok := m["mtu"].(float64)
		if !ok {
			t.Error("interface entry missing or non-numeric 'mtu' field")
		} else if int(mtu) < -1 {
			t.Errorf("interface mtu invalid, got %d", int(mtu))
		}
		// 验证点：必须包含flags字段（接口状态标志，如up|running）
		if flags, ok := m["flags"].(string); !ok || flags == "" {
			t.Error("interface entry missing or empty 'flags' field")
		}
		// 验证点：必须包含mac字段（物理MAC地址）
		if _, ok := m["mac"].(string); !ok {
			t.Error("interface entry missing or non-string 'mac' field")
		}
		// 验证点：必须包含ips字段且为数组，每个子项需有ip/family验证
		ipsRaw, ok := m["ips"].([]interface{})
		if !ok {
			t.Error("interface entry missing or non-array 'ips' field")
			continue
		}
		for _, ipRaw := range ipsRaw {
			ipMap, ok := ipRaw.(map[string]interface{})
			if !ok {
				t.Errorf("ips entry is not a JSON object: %v", ipRaw)
				continue
			}
			if ipStr, ok := ipMap["ip"].(string); !ok || ipStr == "" {
				t.Error("ips entry missing or empty 'ip'")
			} else if net.ParseIP(ipStr) == nil {
				t.Errorf("ips entry 'ip' is not a valid IP: %q", ipStr)
			}
			if fam, ok := ipMap["family"].(string); !ok {
				t.Error("ips entry missing 'family'")
			} else if fam != "v4" && fam != "v6" {
				t.Errorf("ips entry 'family' must be 'v4' or 'v6', got %q", fam)
			}
		}
	}
}

// TestPrintUsage 测试printUsage输出到stderr的内容
// 验证包含Usage关键字和go-engine标识
func TestPrintUsage(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stderr = w

	printUsage()

	w.Close()
	os.Stderr = origStderr

	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read stderr: %v", err)
	}
	output := string(buf)

	if !strings.Contains(output, "Usage:") {
		t.Error("printUsage output missing 'Usage:'")
	}
	if !strings.Contains(output, "go-engine") {
		t.Error("printUsage output missing 'go-engine'")
	}
}

// buildTestBinary 编译go-engine测试二进制文件，返回路径和清理函数
func buildTestBinary(t *testing.T) (string, func()) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "go-engine-test.exe")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join("go-engine")
	// 从当前工作目录推断go-engine目录
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	cmd.Dir = wd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin, func() { os.Remove(bin) }
}

// runTestBinary 运行编译好的二进制并返回stdout、stderr和exit code
func runTestBinary(t *testing.T, bin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec failed: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// TestCmdSendErrors 测试cmdSend参数校验错误路径
// 通过子进程方式测试，因为cmdSend会调用os.Exit(1)
func TestCmdSendErrors(t *testing.T) {
	bin, cleanup := buildTestBinary(t)
	defer cleanup()

	t.Run("missing --peer", func(t *testing.T) {
		_, stderr, code := runTestBinary(t, bin, "send")
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr, "--peer is required") {
			t.Errorf("stderr missing '--peer is required', got: %s", stderr)
		}
	})

	t.Run("missing --file", func(t *testing.T) {
		_, stderr, code := runTestBinary(t, bin, "send", "--peer=127.0.0.1:9527")
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr, "--file is required") {
			t.Errorf("stderr missing '--file is required', got: %s", stderr)
		}
	})
}

// waitForTCPPort 轮询等待TCP端口可连接，超时则fail
func waitForTCPPort(t *testing.T, host string, port int, timeout time.Duration) {
	t.Helper()
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("TCP %s not ready within %v", addr, timeout)
}

// startServeForTest 在后台启动cmdServe，返回ready通道和清理函数
// 使用t.TempDir()作为接收目录，goroutine泄漏在测试结束时由进程退出清理
func startServeForTest(t *testing.T, port int, bindIP string) (ready chan struct{}, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	args := []string{
		fmt.Sprintf("--port=%d", port),
		fmt.Sprintf("--dir=%s", dir),
	}
	if bindIP != "" {
		args = append(args, fmt.Sprintf("--ip=%s", bindIP))
	}

	ready = make(chan struct{})
	go func() {
		// 简单延迟等待mDNS和TCP就绪
		time.Sleep(100 * time.Millisecond)
		close(ready)
		cmdServe(args)
	}()

	// 等待TCP端口就绪（使用绑定IP或默认127.0.0.1）
	host := "127.0.0.1"
	if bindIP != "" {
		host = bindIP
	}
	waitForTCPPort(t, host, port, 5*time.Second)
	// 额外等待mDNS广播传播
	time.Sleep(500 * time.Millisecond)

	return ready, func() {
		// cmdServe阻塞无法优雅关闭，依赖测试进程退出
	}
}

// runDiscoverForTest 运行cmdDiscover并捕获stdout和stderr输出
func runDiscoverForTest(t *testing.T, args []string) (stdout, stderr string) {
	t.Helper()
	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	cmdDiscover(args)

	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	bufOut, err := io.ReadAll(rOut)
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}
	bufErr, err := io.ReadAll(rErr)
	if err != nil {
		t.Fatalf("Failed to read stderr: %v", err)
	}
	return string(bufOut), string(bufErr)
}

// parseDiscoverOutput 解析discover命令的JSON输出，返回peers数组
func parseDiscoverOutput(t *testing.T, output string) []map[string]interface{} {
	t.Helper()
	t.Logf("raw discover output: %s", output)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("discover output is not valid JSON: %v\nOutput: %s", err, output)
	}
	if typ, ok := parsed["type"].(string); !ok || typ != "peers" {
		t.Fatalf("expected type=peers, got %v", parsed["type"])
	}
	peersRaw, ok := parsed["peers"].([]interface{})
	if !ok {
		// peers为null时返回空切片
		return nil
	}
	var peers []map[string]interface{}
	for _, raw := range peersRaw {
		m, ok := raw.(map[string]interface{})
		if !ok {
			t.Errorf("peer entry is not a JSON object: %v", raw)
			continue
		}
		peers = append(peers, m)
	}
	return peers
}

// TestServeAndDiscover 基础集成测试：启动serve后通过discover发现
// 验证mDNS注册→发现完整流程
func TestServeAndDiscover(t *testing.T) {
	const port = 19530
	startServeForTest(t, port, "")

	output, stderr := runDiscoverForTest(t, []string{"--timeout=3"})
	if stderr != "" {
		t.Logf("discover stderr: %s", stderr)
	}
	peers := parseDiscoverOutput(t, output)

	t.Logf("Discovered %d peer(s)", len(peers))

	found := false
	for _, p := range peers {
		pPort, _ := p["port"].(float64)
		if int(pPort) == port {
			found = true
			if addr, ok := p["addr"].(string); !ok || addr == "" {
				t.Error("discovered peer 'addr' is empty")
			}
			if host, ok := p["host"].(string); !ok || host == "" {
				t.Error("discovered peer 'host' is empty")
			}
			if name, ok := p["name"].(string); !ok || name == "" {
				t.Error("discovered peer 'name' is empty")
			}
			t.Logf("Found peer: addr=%v host=%v port=%v name=%v",
				p["addr"], p["host"], pPort, p["name"])
			break
		}
	}
	if !found {
		t.Errorf("peer with port %d not found among %d discovered peers", port, len(peers))
	}
}

// TestServeAndDiscover_BindIP 绑定指定IP的集成测试
// 自动检测可用IPv4地址，验证peer的host与绑定IP一致
func TestServeAndDiscover_BindIP(t *testing.T) {
	ips, err := discovery.ListIPs()
	if err != nil {
		t.Fatalf("ListIPs failed: %v", err)
	}
	var testIP string
	for _, info := range ips {
		t.Logf("available IP: %s (%s, %s)", info.IP, info.Iface, info.Family)
		if info.Family == "v4" && testIP == "" {
			testIP = info.IP
		}
	}
	if testIP == "" {
		t.Skip("no IPv4 address available for bind-IP test")
	}
	t.Logf("selected test IP: %s", testIP)

	const port = 19531
	startServeForTest(t, port, testIP)

	output, stderr := runDiscoverForTest(t, []string{"--timeout=3"})
	if stderr != "" {
		t.Logf("discover stderr: %s", stderr)
	}
	peers := parseDiscoverOutput(t, output)

	t.Logf("Discovered %d peer(s), looking for bound IP %s", len(peers), testIP)

	// 验证至少有一个peer的host与绑定IP一致（mDNS可能发现多个同主机peer）
	found := false
	for _, p := range peers {
		host, _ := p["host"].(string)
		if host == testIP {
			found = true
			t.Logf("Found peer on bound IP: addr=%v host=%v port=%v",
				p["addr"], p["host"], p["port"])
			break
		}
	}
	if !found {
		t.Errorf("no peer with host %s found among %d discovered peers", testIP, len(peers))
	}
}

// TestServeAndDiscover_Multiple 多实例集成测试
// 同时启动2个serve，通过单次discover验证serve正常工作
// 注意：mDNS使用os.Hostname()注册，同主机多实例会被去重，仅最后一个可见
func TestServeAndDiscover_Multiple(t *testing.T) {
	const port1 = 19532
	const port2 = 19533

	// 并行启动两个serve
	go func() {
		dir := t.TempDir()
		cmdServe([]string{
			fmt.Sprintf("--port=%d", port1),
			fmt.Sprintf("--dir=%s", dir),
		})
	}()
	go func() {
		dir := t.TempDir()
		cmdServe([]string{
			fmt.Sprintf("--port=%d", port2),
			fmt.Sprintf("--dir=%s", dir),
		})
	}()

	// 等待两个端口就绪
	waitForTCPPort(t, "127.0.0.1", port1, 5*time.Second)
	waitForTCPPort(t, "127.0.0.1", port2, 5*time.Second)
	time.Sleep(500 * time.Millisecond)

	output, stderr := runDiscoverForTest(t, []string{"--timeout=4"})
	if stderr != "" {
		t.Logf("discover stderr: %s", stderr)
	}
	peers := parseDiscoverOutput(t, output)

	t.Logf("Discovered %d peer(s)", len(peers))

	// mDNS同主机多实例会被去重，验证至少发现一个peer
	if len(peers) == 0 {
		t.Fatal("expected at least 1 peer, got 0")
	}

	for _, p := range peers {
		t.Logf("Found peer: addr=%v host=%v port=%v", p["addr"], p["host"], p["port"])
	}
}
