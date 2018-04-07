package diskqueue

import (
	"context"
	"net"
	"runtime"
	"strings"

	"github.com/tddhit/tools/log"
)

type TCPServer struct {
	listener net.Listener
	address  string
}

func NewTCPServer(address string) *TCPServer {
	s := &TCPServer{
		address: address,
	}
	return s
}

func (s *TCPServer) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				log.Warnf("temporary Accept() failure - %s", err)
				runtime.Gosched()
				continue
			}
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("listener.Accept() - %s", err)
			}
			break
		}
		go s.Handle(ctx, clientConn)
	}
	return nil
}

func (s *TCPServer) Handle(ctx context.Context, clientConn net.Conn) {
	prot := &protocol{}
	if err := prot.IOLoop(ctx, clientConn); err != nil {
		log.Errorf("client(%s) - %s", clientConn.RemoteAddr(), err)
		return
	}
}
