// main包的单元测试
// 测试命令行参数解析和CLI命令输出
package main

import (
	"encoding/json"
	"io"
	"os"
	"testing"
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
		if _, ok := m["ip"]; !ok {
			t.Error("IP entry missing 'ip' field")
		}
		// 验证点：每个IP条目必须包含iface字段（网络接口名）
		if _, ok := m["iface"]; !ok {
			t.Error("IP entry missing 'iface' field")
		}
		// 验证点：每个IP条目必须包含family字段（地址族v4/v6）
		if _, ok := m["family"]; !ok {
			t.Error("IP entry missing 'family' field")
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
		if _, ok := m["name"]; !ok {
			t.Error("interface entry missing 'name' field")
		}
		// 验证点：必须包含index字段（接口索引号）
		if _, ok := m["index"]; !ok {
			t.Error("interface entry missing 'index' field")
		}
		// 验证点：必须包含mtu字段（最大传输单元）
		if _, ok := m["mtu"]; !ok {
			t.Error("interface entry missing 'mtu' field")
		}
		// 验证点：必须包含flags字段（接口状态标志，如up|running）
		if _, ok := m["flags"]; !ok {
			t.Error("interface entry missing 'flags' field")
		}
		// 验证点：必须包含mac字段（物理MAC地址）
		if _, ok := m["mac"]; !ok {
			t.Error("interface entry missing 'mac' field")
		}
		// 验证点：必须包含ips字段（绑定的IP地址列表）
		if _, ok := m["ips"]; !ok {
			t.Error("interface entry missing 'ips' field")
		}
	}
}
