package main

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/websocket"
)

// jinfo "ws://10.0.6.54:8188/janus"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: jinfo <wsaddr>\n")
		return
	}

	conn, err := websocket.Dial(os.Args[1], "janus-protocol", "http://10.0.6.54")
	if err != nil {
		fmt.Printf("Connect: %s\n", err)
		return
	}
	defer conn.Close()

	// Create 'info' request
	info := make(map[string]string)
	info["janus"] = "info"
	info["transaction"] = "1234"

	// Marshal request to json and sent to Janus
	req, _ := json.Marshal(info)
	conn.Write(req)

	// Receive response
	res := make([]byte, 10240)
	n, _ := conn.Read(res)
	_ = n

	fmt.Printf("Received: %s\n", string(res))
}
