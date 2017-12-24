package proto

import "fmt"

type ClientHello struct {
	version   byte //version, 5
	methodLen byte
	methods   []byte // authentication methods supported by client.
}

type ServerHello struct {
	version byte
	method  byte
}

type ClientCommand struct {
	Version     byte
	Command     byte
	reserved    byte
	AddressType byte // 1: ipv4, length: 4 ;3:domain name, length:first byte;4: ipv6 length: 16
	DestAddr    string
	Port        uint16
}
type ServerResponse struct {
	version     byte
	reply       byte //connect status: 0, success;1
	reserved    byte
	addressType byte
	bindAddr    []byte
	port        uint16
}

func DecodeClientHello(buffer []byte) (*ClientHello, error) {
	if len(buffer) < 3 {
		return nil, fmt.Errorf("error parsing ClientHello")
	}
	var clientHello = ClientHello{version: buffer[0], methodLen: buffer[1], methods: buffer[2 : 2+buffer[1]]}
	return &clientHello, nil
}

func DecodeClientCommand(buffer []byte) (*ClientCommand, error) {
	if len(buffer) < 5 {
		return nil, fmt.Errorf("error parsing ClientHello")
	}
	var command = ClientCommand{Version: buffer[0], Command: buffer[1], reserved: buffer[2], AddressType: buffer[3]}

	//default set to ipv4
	var begin = 4
	var addrLen = 4
	if command.AddressType == 1 { //ipv4
		//default
		command.DestAddr = fmt.Sprintf("%d.%d.%d.%d", buffer[begin], buffer[begin+1],
			buffer[begin+2], buffer[begin+3])

	} else if command.AddressType == 3 { // domain_name
		begin = 4 + 1
		addrLen = int(buffer[4])
		command.DestAddr = string(buffer[begin : begin+addrLen])

	} else if command.AddressType == 4 { //ipv6
		addrLen = 16
		command.DestAddr = fmt.Sprintf("%v", buffer[begin:begin+16])
	}

	offset := begin + addrLen
	command.Port = (uint16(buffer[offset]))<<8 + uint16(buffer[offset+1])
	return &command, nil
}
