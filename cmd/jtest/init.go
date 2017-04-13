package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	WsAddr   string `long:"ws" default:"ws://10.0.6.22:8188/janus" description:"WebSocket address"`
	LogLevel string `long:"log_level" default:"info" description:"log level"`
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func init() {
	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash|flags.IgnoreUnknown)

	_, err := parser.Parse()
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(-1)
	}
}

func init() {
	if level, err := log.ParseLevel(strings.ToLower(opts.LogLevel)); err != nil {
		log.SetLevel(level)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}
