package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/randutil"
	"github.com/keroserene/go-webrtc"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/net/websocket"
)

var transMap map[string]chan []byte
var gsessionID int64
var ghandleID int64
var gsdp chan string

func init() {
	transMap = make(map[string]chan []byte)
	gsdp = make(chan string, 1)
}

var pc *webrtc.PeerConnection
var dc *webrtc.DataChannel

func processRecv(conn *websocket.Conn) {
	buffer := make([]byte, 10240)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Errorf("conn.Read: %s\n", err)
			return
		}

		log.Infof("Signal receive: %v", string(buffer[:n]))

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
		} else if typ == "timeout" {
			sessionID := gjson.GetBytes(buffer[:n], "session_id").Int()
			log.Infof("timeout session_id: %v", sessionID)
		} else if typ == "ack" {
			transaction := gjson.GetBytes(buffer[:n], "transaction").String()

			log.Infof("type: %v, transaction: %v", typ, transaction)
			if ch, ok := transMap[transaction]; ok {
				ch <- buffer[:n]
			}
		} else if typ == "event" {
			transaction := gjson.GetBytes(buffer[:n], "transaction").String()

			log.Infof("type: %v, transaction: %v", typ, transaction)
			if ch, ok := transMap[transaction]; ok {
				ch <- buffer[:n]
			}
		} else {
			transaction := gjson.GetBytes(buffer[:n], "transaction").String()

			log.Infof("type: %v, transaction: %v", typ, transaction)
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

// {"janus":"attach","session_id":1138646217789133, "plugin":"janus.plugin.echotest","opaque_id":"echotest-IYkWieIc8SUq","transaction":"s785S3V7wWnj"}
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

	ghandleID = gjson.GetBytes(bytes, "data.id").Int()

	return nil
}

// {"janus":"message","body":{"audio":false,"video":false},"transaction":"R1y2XvCeze7S", "jsep":{"type":"offer", "sdp":""}, "session_id":1138646217789133, "handle_id": 594589210486401}
func sendMessageSig(conn *websocket.Conn) error {
	msg := []byte(`{"janus":"message","body":{"audio":false,"video":false},"transaction":"R1y2XvCeze7S","jsep":{"type":"offer", "sdp":""},"session_id":1138646217789133, "handle_id": 594589210486401}`)
	trans, _ := randutil.AlphaString(12)

	req, _ := sjson.SetBytes(msg, "transaction", trans)
	req, _ = sjson.SetBytes(req, "session_id", gsessionID)
	req, _ = sjson.SetBytes(req, "handle_id", ghandleID)
	sdp := <-gsdp
	sdpContent := gjson.Get(sdp, "sdp").String()
	log.Infof("generated sdp: %v", sdpContent)
	req, _ = sjson.SetBytes(req, "jsep.sdp", sdpContent)
	log.Infof("Message signal request: %v", string(req))

	_, err := conn.Write(req)
	if err != nil {
		return err
	}

	response := make(chan []byte)
	transMap[trans] = response

	bytes := <-response
	log.Infof("Message ack response: %v", string(bytes))

	bytes = <-response
	log.Infof("Message event response: %v", string(bytes))

	answer := gjson.GetBytes(bytes, "jsep.sdp").String()
	ansserSDP := webrtc.DeserializeSessionDescription()

	return nil
}

// {"janus":"keepalive", "session_id":1138646217789133, "transaction":"sBJNyUhH6Vc6"}
func sendKeepalive(conn *websocket.Conn) {
	msg := []byte(`{"janus":"keepalive", "session_id":1138646217789133, "transaction":"sBJNyUhH6Vc6"}`)

	for {
		if gsessionID != 0 {
			trans, _ := randutil.AlphaString(12)
			req, _ := sjson.SetBytes(msg, "transaction", trans)
			req, _ = sjson.SetBytes(req, "session_id", gsessionID)
			log.Infof("Keepalive signal request: %v", string(req))

			_, err := conn.Write(req)
			if err != nil {
				return
			}

			response := make(chan []byte)
			transMap[trans] = response

			bytes := <-response
			log.Infof("Keepalive response: %v", string(bytes))
		}
		time.Sleep(30 * time.Second)
	}
}

//
// Preparing SDP messages for signaling.
// generateOffer and generateAnswer are expected to be called within goroutines.
// It is possible to send the serialized offers or answers immediately upon
// creation, followed by subsequent individual ICE candidates.
//
// However, to ease the user's copy & paste experience, in this case we forgo
// the trickle ICE and wait for OnIceComplete to fire, which will contain
// a full SDP mesasge with all ICE candidates, so the user only has to copy
// one message.
//

func generateOffer() {
	fmt.Println("Generating offer...")
	offer, err := pc.CreateOffer() // blocking
	if err != nil {
		fmt.Println(err)
		return
	}
	pc.SetLocalDescription(offer)
}

// Attach callbacks to a newly created data channel.
// In this demo, only one data channel is expected, and is only used for chat.
// But it is possible to send any sort of bytes over a data channel, for many
// more interesting purposes.
func prepareDataChannel(channel *webrtc.DataChannel) {
	channel.OnOpen = func() {
		log.Infoln("Data Channel Opened!")
		//startChat()
	}
	channel.OnClose = func() {
		log.Infoln("Data Channel closed.")
		//endChat()
	}
	channel.OnMessage = func(msg []byte) {
		log.Infoln("Data Channel message received.")
		//receiveChat(string(msg))
	}
}

// Create a PeerConnection.
// If |instigator| is true, create local data channel which causes a
// negotiation-needed, leading to preparing an SDP offer to be sent to the
// remote peer. Otherwise, await an SDP offer from the remote peer, and send an
// answer back.
func start(instigator bool) {
	var err error
	log.Infoln("Starting up PeerConnection...")
	// TODO: Try with TURN servers.
	config := webrtc.NewConfiguration(
		webrtc.OptionIceServer("stun:stun.ekiga.net:3478"))

	pc, err = webrtc.NewPeerConnection(config)
	if nil != err {
		log.Errorln("Failed to create PeerConnection.")
		return
	}

	// OnNegotiationNeeded is triggered when something important has occurred in
	// the state of PeerConnection (such as creating a new data channel), in which
	// case a new SDP offer must be prepared and sent to the remote peer.
	pc.OnNegotiationNeeded = func() {
		go generateOffer()
	}
	// Once all ICE candidates are prepared, they need to be sent to the remote
	// peer which will attempt reaching the local peer through NATs.
	pc.OnIceComplete = func() {
		log.Infoln("Finished gathering ICE candidates.")
		sdp := pc.LocalDescription().Serialize()
		gsdp <- sdp
	}
	/*
		pc.OnIceGatheringStateChange = func(state webrtc.IceGatheringState) {
			fmt.Println("Ice Gathering State:", state)
			if webrtc.IceGatheringStateComplete == state {
				// send local description.
			}
		}
	*/
	// A DataChannel is generated through this callback only when the remote peer
	// has initiated the creation of the data channel.
	pc.OnDataChannel = func(channel *webrtc.DataChannel) {
		log.Infoln("Datachannel established by remote... ", channel.Label())
		dc = channel
		prepareDataChannel(channel)
	}

	if instigator {
		// Attempting to create the first datachannel triggers ICE.
		log.Infoln("Initializing datachannel....")
		dc, err = pc.CreateDataChannel("test", webrtc.Init{})
		if nil != err {
			log.Errorln("Unexpected failure creating Channel.")
			return
		}
		prepareDataChannel(dc)
	}
}

func main() {
	webrtc.SetLoggingVerbosity(1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		fmt.Println("Demo interrupted. Disconnecting...")
		if nil != dc {
			dc.Close()
		}
		if nil != pc {
			pc.Close()
		}
		os.Exit(1)
	}()

	conn, err := websocket.Dial(opts.WsAddr, "janus-protocol", "http://10.0.6.22")
	if err != nil {
		fmt.Printf("Connect: %s\n", err)
		return
	}
	defer conn.Close()

	go processRecv(conn)

	sendCreateSig(conn)

	go sendKeepalive(conn)

	sendAttachSig(conn)

	start(true)

	sendMessageSig(conn)

	select {}
}
