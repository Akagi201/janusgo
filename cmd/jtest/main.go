package main

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/randutil"
	"github.com/tidwall/gjson"
	"golang.org/x/net/websocket"
)

var transMap map[string]chan []byte
var gsessionID int64

func init() {
	transMap = make(map[string]chan []byte)
}

//var pc *webrtc.PeerConnection
//var dc *webrtc.DataChannel

func processRecv(conn *websocket.Conn) {
	buffer := make([]byte, 10240)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Errorf("conn.Read: %s\n", err)
			return
		}

		log.Infof("Create signal receive: %v", string(buffer[:n]))

		typ := gjson.GetBytes(buffer[:n], "janus").String()
		if typ == "event" {
			sessionID := gjson.GetBytes(buffer[:n], "session_id").Int()
			transaction := gjson.GetBytes(buffer[:n], "transaction").String()
			sender := gjson.GetBytes(buffer[:n], "sender").Int()

			log.Infof("type: %v, sessionID: %v, transaction: %v, sender: %v", typ, sessionID, transaction, sender)
			if ch, ok := transMap[transaction]; ok {
				ch <- buffer[:n]
			}
		} else if typ == "success" {
			transaction := gjson.GetBytes(buffer[:n], "transaction").String()
			sessionID := gjson.GetBytes(buffer[:n], "data.id").Int()

			log.Infof("type: %v, sessionID: %v, transaction: %v", typ, sessionID, transaction)
			if ch, ok := transMap[transaction]; ok {
				ch <- buffer[:n]
			}
		}
	}
}

// {"janus":"create", "transaction":"123456"}
func sendCreateSig(conn *websocket.Conn) error {
	createSig := make(map[string]string)
	createSig["janus"] = "create"
	createSig["transaction"], _ = randutil.AlphaString(12)

	// Marshal request to json and sent to Janus
	req, _ := json.Marshal(createSig)
	log.Infof("create signal request: %v", string(req))
	_, err := conn.Write(req)
	if err != nil {
		return err
	}

	response := make(chan []byte)

	transMap[createSig["transaction"]] = response

	bytes := <-response

	log.Infof("Create response: %v", string(bytes))

	gsessionID = gjson.GetBytes(bytes, "data.id").Int()

	return nil
}

// {"janus":"attach","session_id":1145173870808008, "plugin":"janus.plugin.echotest","opaque_id":"echotest-IYkWieIc8SUq","transaction":"s785S3V7wWnj"}
func sendAttachSig(conn *websocket.Conn) error {
	attachSig := make(map[string]interface{})
	attachSig["janus"] = "attach"
	attachSig["session_id"] = gsessionID
	attachSig["plugin"] = "janus.plugin.echotest"
	attachSig["transaction"], _ = randutil.AlphaString(12)
	attachSig["opaque_id"] = "echotest-" + attachSig["transaction"].(string) // strings.Join([]string{"echotest-", attachSig["transaction"].(string)}, "")

	req, _ := json.Marshal(attachSig)
	log.Infof("create signal request: %v", string(req))

	_, err := conn.Write(req)
	if err != nil {
		return err
	}

	response := make(chan []byte)
	transMap[attachSig["transaction"].(string)] = response

	bytes := <-response
	log.Infof("Attach response: %v", string(bytes))

	return nil
}

func main() {
	//sigs := make(chan os.Signal, 1)
	//signal.Notify(sigs, os.Interrupt)
	//go func() {
	//	<-sigs
	//	fmt.Println("Demo interrupted. Disconnecting...")
	//	if nil != dc {
	//		dc.Close()
	//	}
	//	if nil != pc {
	//		pc.Close()
	//	}
	//	os.Exit(1)
	//}()

	conn, err := websocket.Dial(opts.WsAddr, "janus-protocol", "http://10.0.6.22")
	if err != nil {
		fmt.Printf("Connect: %s\n", err)
		return
	}
	defer conn.Close()

	go processRecv(conn)

	sendCreateSig(conn)

	sendAttachSig(conn)
}
