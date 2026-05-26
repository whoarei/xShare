package main

import (
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestGetArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		key      string
		defVal   string
		expected string
	}{
		{
			name:     "exact match",
			args:     []string{"--port=9527", "--dir=./received"},
			key:      "port",
			defVal:   "",
			expected: "9527",
		},
		{
			name:     "match with prefix",
			args:     []string{"--port-num=8080", "--port=9527"},
			key:      "port",
			defVal:   "",
			expected: "9527",
		},
		{
			name:     "not found returns default",
			args:     []string{"--port=9527"},
			key:      "dir",
			defVal:   "./received",
			expected: "./received",
		},
		{
			name:     "empty args returns default",
			args:     []string{},
			key:      "timeout",
			defVal:   "5",
			expected: "5",
		},
		{
			name:     "ip argument (v4)",
			args:     []string{"--ip=192.168.1.100", "--timeout=3"},
			key:      "ip",
			defVal:   "",
			expected: "192.168.1.100",
		},
		{
			name:     "ip argument (v6)",
			args:     []string{"--ip=2001:db8::1", "--port=8080"},
			key:      "ip",
			defVal:   "",
			expected: "2001:db8::1",
		},
		{
			name:     "empty value falls back to default",
			args:     []string{"--port="},
			key:      "port",
			defVal:   "9527",
			expected: "9527",
		},
		{
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

func TestCmdListIPs(t *testing.T) {
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

	if output == "" {
		t.Fatal("Expected non-empty stdout output")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Stdout is not valid JSON: %v\nOutput: %s", err, output)
	}

	if typ, ok := parsed["type"].(string); !ok || typ != "ips" {
		t.Errorf("Expected type=ips, got %v", parsed["type"])
	}

	ips, ok := parsed["ips"].([]interface{})
	if !ok {
		t.Fatal("ips field missing or not an array")
	}

	t.Logf("Listed %d IP(s)", len(ips))

	for _, raw := range ips {
		m, ok := raw.(map[string]interface{})
		if !ok {
			t.Errorf("IP entry is not a JSON object: %v", raw)
			continue
		}
		if _, ok := m["ip"]; !ok {
			t.Error("IP entry missing 'ip' field")
		}
		if _, ok := m["iface"]; !ok {
			t.Error("IP entry missing 'iface' field")
		}
		if _, ok := m["family"]; !ok {
			t.Error("IP entry missing 'family' field")
		}
	}
}

func TestCmdListIfaces(t *testing.T) {
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

	if output == "" {
		t.Fatal("Expected non-empty stdout output")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Stdout is not valid JSON: %v\nOutput: %s", err, output)
	}

	if typ, ok := parsed["type"].(string); !ok || typ != "interfaces" {
		t.Errorf("Expected type=interfaces, got %v", parsed["type"])
	}

	ifaces, ok := parsed["interfaces"].([]interface{})
	if !ok {
		t.Fatal("interfaces field missing or not an array")
	}

	t.Logf("Listed %d interface(s)", len(ifaces))

	if len(ifaces) == 0 {
		t.Fatal("expected at least one network interface")
	}

	for _, raw := range ifaces {
		m, ok := raw.(map[string]interface{})
		if !ok {
			t.Errorf("interface entry is not a JSON object: %v", raw)
			continue
		}
		if _, ok := m["name"]; !ok {
			t.Error("interface entry missing 'name' field")
		}
		if _, ok := m["index"]; !ok {
			t.Error("interface entry missing 'index' field")
		}
		if _, ok := m["mtu"]; !ok {
			t.Error("interface entry missing 'mtu' field")
		}
		if _, ok := m["flags"]; !ok {
			t.Error("interface entry missing 'flags' field")
		}
		if _, ok := m["mac"]; !ok {
			t.Error("interface entry missing 'mac' field")
		}
		if _, ok := m["ips"]; !ok {
			t.Error("interface entry missing 'ips' field")
		}
	}
}
