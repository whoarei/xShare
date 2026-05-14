package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"go-engine/pkg/discovery"
	"go-engine/pkg/transfer"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "discover":
		cmdDiscover(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `xShare v1 - LAN file sharing tool

Usage:
  go-engine serve --port=PORT [--dir=DIR]
    Start the server to receive files.

  go-engine discover [--timeout=SECONDS]
    Discover peers on the local network.

  go-engine send --peer=ADDR --dir=DIR
    Send a directory to a peer.

Options:
  --port=PORT       TCP port (default: 9527)
  --dir=DIR         Target directory (default: ./received or current dir)
  --peer=ADDR       Peer address in host:port format
  --timeout=SECONDS Discovery timeout in seconds (default: 5)
`)
}

func getArg(args []string, name string, defaultVal string) string {
	prefix := "--" + name + "="
	for _, a := range args {
		if len(a) > len(prefix) && a[:len(prefix)] == prefix {
			return a[len(prefix):]
		}
	}
	return defaultVal
}

func cmdServe(args []string) {
	port := 9527
	portStr := getArg(args, "port", "")
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	dir := getArg(args, "dir", "./received")
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}

	srv, err := discovery.Register(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}
	defer srv.Shutdown()

	fmt.Printf(`{"type":"ready","port":%d,"dir":"%s"}`+"\n", port, absDir)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	errCh := make(chan error, 1)
	go func() {
		errCh <- startTCPServer(port, absDir)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		}
	case <-sigCh:
		fmt.Println(`{"type":"shutdown"}`)
	}
}

func startTCPServer(port int, targetDir string) error {
	receiver := transfer.NewReceiver(targetDir)
	return receiver.ListenAndServe(port)
}

func cmdDiscover(args []string) {
	timeoutStr := getArg(args, "timeout", "5")
	timeoutSec := 5
	fmt.Sscanf(timeoutStr, "%d", &timeoutSec)

	peers, err := discovery.Discover(time.Duration(timeoutSec) * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}

	result, _ := json.Marshal(map[string]interface{}{
		"type":  "peers",
		"peers": peers,
	})
	fmt.Println(string(result))
}

func cmdSend(args []string) {
	peer := getArg(args, "peer", "")
	if peer == "" {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"--peer is required"}`+"\n")
		os.Exit(1)
	}

	dir := getArg(args, "dir", "")
	if dir == "" {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"--dir is required"}`+"\n")
		os.Exit(1)
	}

	sender, err := transfer.NewSender(peer)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}
	defer sender.Close()

	if err := sender.SendDirectory(dir); err != nil {
		fmt.Fprintf(os.Stderr, `{"type":"error","msg":"%s"}`+"\n", err.Error())
		os.Exit(1)
	}
}
