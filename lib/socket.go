package ircb

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
)

type Commander struct{}

func (c *Commander) Echo(request, reply *string) error {
	*reply = *request
	return nil
}

func SocketInit(logger *log.Logger) error {
	commander := new(Commander)
	rpc.Register(commander)
	rpc.HandleHTTP()
	l, e := net.Listen("unix", "control.socket")
	if e != nil {
		return e
	}
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	go ListenSocket(l, logger)
	return nil
}

func ListenSocket(l net.Listener, logger *log.Logger) error {
	conn, err := l.Accept()
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return fmt.Errorf("closed properly")
		}

		return fmt.Errorf("diamond: Could not accept connection: %v",
			err)

	}
	rpcServer := rpc.NewServer()
	var pack = new(Commander)

	if err = rpcServer.RegisterName("Packet", pack); err != nil {
		return fmt.Errorf("diamond: %s",
			err.Error())
	}
	go func() {
		if conn != nil {
			log.Println("Got conn:", conn.LocalAddr().String())
		}
		rpcServer.ServeConn(conn)
		conn.Close()
	}()

	return nil
}
