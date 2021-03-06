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
	Version  byte
	Command  byte
	reserved byte
	// 1: ipv4, length: 4 ;
	// 3:domain name, length:first byte;
	// 4: ipv6 length: 16
	AddressType byte
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

/**
 * considering most of client commands size is less than MTU(1500),
 * any command which is longer than 1024 will be ignored
 */
func DecodeClientCommand(buffer []byte) (*ClientCommand, error) {
	total := len(buffer)
	if total < 5 {
		return nil, fmt.Errorf("client command packet is too short")
	}
	var command = ClientCommand{Version: buffer[0], Command: buffer[1], reserved: buffer[2], AddressType: buffer[3]}

	//default set to ipv4
	begin := 4
	checkReadOverFlow := func(addrLen int) error {
		portLen := 2
		lenWanted := begin + addrLen + portLen
		if lenWanted > total {
			//buffer overflow
			return fmt.Errorf("malformed packet, wanted length: %d, actually: %d", lenWanted, total)
		}
		return nil
	}

	var addrLen = 0
	if command.AddressType == 1 { //ipv4
		//default
		addrLen = 4
		if err := checkReadOverFlow(addrLen); err != nil {
			return nil, err
		}
		command.DestAddr = fmt.Sprintf("%d.%d.%d.%d", buffer[begin], buffer[begin+1],
			buffer[begin+2], buffer[begin+3])
	} else if command.AddressType == 3 { // domain_name
		domainNameLen := int(buffer[4]) //1-255
		if domainNameLen == 0 {
			return nil, fmt.Errorf("domain name length cannot be 0")
		}
		addrLen = domainNameLen + 1
		if err := checkReadOverFlow(addrLen); err != nil {
			return nil, err
		}
		command.DestAddr = string(buffer[begin+1 : begin+1+domainNameLen])
	} else if command.AddressType == 4 { //ipv6
		addrLen = 16
		if err := checkReadOverFlow(addrLen + 1); err != nil {
			return nil, err
		}
		command.DestAddr = fmt.Sprintf("%v", buffer[begin:begin+16])
	} else {
		return nil, fmt.Errorf("invalid address type %d", command.AddressType)
	}

	offset := begin + addrLen
	command.Port = (uint16(buffer[offset]))<<8 + uint16(buffer[offset+1])
	return &command, nil
}
