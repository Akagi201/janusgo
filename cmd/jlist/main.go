package main

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/websocket"
)

// jinfo "ws://10.0.6.54:7188/admin"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: jlist <wsaddr> [admin_secret]\n")
		return
	}

	conn, err := websocket.Dial(os.Args[1], "janus-admin-protocol", "http://10.0.6.54")
	if err != nil {
		fmt.Printf("Connect: %s\n", err)
		return
	}
	defer conn.Close()

	// Create 'list_sessions' request
	list_sessions := make(map[string]string)
	list_sessions["janus"] = "list_sessions"
	list_sessions["transaction"] = "1234"
	list_sessions["admin_secret"] = "janusoverlord"
	if len(os.Args) > 2 {
		list_sessions["admin_secret"] = os.Args[2]
	}

	// Marshal request to json and sent to Janus
	req, _ := json.Marshal(list_sessions)
	conn.Write(req)

	// Receive response
	res := make([]byte, 8192)
	n, _ := conn.Read(res)
	_ = n

	fmt.Printf("Received: %s\n", string(res))

	// Format output
	//var out bytes.Buffer
	//json.Indent(&out, res[:n], "", "\t")
	//out.Write([]byte("\n"))

	// Write to stdout
	//out.WriteTo(os.Stdout)
}
