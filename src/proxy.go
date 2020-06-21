package main

import (
	"flag"
	"github.com/Matrix0xCC/gotunnel/chassis"
	"log"
)

func initProxy() *chassis.Config {
	var config = new(chassis.Config)

	flag.StringVar(&config.Mode, "m", "client",
		"running Mode. client vs server. default to client")

	flag.StringVar(&config.Listen, "l", "127.0.0.1:9000",
		"Listen address. used to specified Listen address in client or server Mode")

	flag.StringVar(&config.Connect, "c", "127.0.0.1:9090",
		"Connect address. only used in client Mode")

	flag.BoolVar(&config.Secure, "s", false,
		"use tls between client and server")

	flag.Parse()

	if config.Mode != "client" && config.Mode != "server" {
		log.Fatalf("invalid Mode %s. only client and server Mode supported.", config.Mode)
	}

	return config
}

func main() {
	config := initProxy()

	log.Printf("start using Config: %+v", config)

	var server chassis.Server
	if config.Mode == "client" {
		server = chassis.NewLocalServer(config)
	} else {
		server = chassis.NewRemoteServer(config)
	}
	server.MainLoop()
}
