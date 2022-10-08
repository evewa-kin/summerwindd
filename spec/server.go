package spec

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/summerwind/h2spec/config"
	"github.com/summerwind/h2spec/log"
	"golang.org/x/net/http2"
)

type Server struct {
	listeners []net.Listener
	config    *config.Config
	spec      *ClientTestGroup
}

func Listen(c *config.Config, tg *ClientTestGroup) (*Server, error) {
	testCases := make(map[int]*ClientTestCase)
	tg.ClientTestCases(testCases, c.FromPort)

	server := &Server{
		listeners: make([]net.Listener, 0),
		config:    c,
	}

	for port, tc := range testCases {
		var err error
		var listener net.Listener

		addr := fmt.Sprintf("%s:%d", c.Host, port)

		if c.TLS {
			tlsConfig, err := c.TLSConfig()
			if err != nil {
				return nil, err
			}

			listener, err = tls.Listen("tcp", addr, tlsConfig)
			if err != nil {
				return nil, err
			}
		} else {
			listener, err = net.Listen("tcp", addr)
			if err != nil {
				return nil, err
			}
		}

		server.listeners = append(server.listeners, listener)
		go server.RunListener(listener, tc)
	}

	return server, nil
}

func (server *Server) RunListener(listener net.Listener, tc *ClientTestCase) {
	for {
		baseConn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		conn, err := Accept(server.config, baseConn)
		if err != nil {
			log.Println(err)
			continue
		}

		go server.handleConn(conn, tc)
	}
}

func (server *Server) Close() {
	for _, listener := range server.listeners {
		listener.Close()
	}
}

func (server *Server) handleConn(conn *Conn, tc *ClientTestCase) {
	start := time.Now()

	err := conn.Handshake()
	if err != nil {
		log.Println(red(err))
		return
	}
	request, err := conn.ReadRequest()
	if err != nil {
		log.Println(red(err))
		return
	}

	err = tc.Run(server.config, conn, request)
	end := time.Now()

	// Ensure that connection had been closed
	go closeConn(conn, request.StreamID)

	log.ResetLine()

	tr := NewClientTestResult(tc, err, end.Sub(start))
	tr.Print()

	if tc.Result != nil {
		tc.Parent.IncRecursive(tc.Result.Failed, tc.Result.Skipped, -1)
	}

	tc.Result = tr
	tc.Parent.IncRecursive(tc.Result.Failed, tc.Result.Skipped, 1)
}

func closeConn(conn *Conn, lastStreamID uint32) {
	if !conn.Closed {
		conn.WriteGoAway(lastStreamID, http2.ErrCodeNo, make([]byte, 0))
		time.Sleep(1 * time.Second)
	}

	conn.Close()
}
