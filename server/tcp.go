package diskqueue

import (
	"context"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/tddhit/tools/log"
)

type TCPServer struct {
	sync.RWMutex

	listener net.Listener
	address  string

	conns []net.Conn

	wg sync.WaitGroup
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
		s.wg.Add(1)
		log.Info("Client Add.")
		go func() {
			s.Handle(ctx, clientConn)
			s.wg.Done()
			log.Info("Client Done.")
		}()
	}
	return nil
}

func (s *TCPServer) Handle(ctx context.Context, clientConn net.Conn) {
	prot := &protocol{}

	s.Lock()
	s.conns = append(s.conns, clientConn)
	s.Unlock()

	if err := prot.IOLoop(ctx, clientConn); err != nil {
		log.Errorf("client(%s) - %s", clientConn.RemoteAddr(), err)
		return
	}
}

func (s *TCPServer) Close() {
	s.listener.Close()

	s.Lock()
	for _, c := range s.conns {
		c.Close()
	}
	s.Unlock()
}
