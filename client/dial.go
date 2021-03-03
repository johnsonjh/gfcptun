package main

import (
	kcp "github.com/johnsonjh/gfcp"
	"github.com/pkg/errors"
	"github.com/xtaci/tcpraw"
)

func dial(config *Config) (*kcp.UDPSession, error) {
	if config.TCP {
		conn, err := tcpraw.Dial("tcp", config.RemoteAddr)
		if err != nil {
			return nil, errors.Wrap(err, "tcpraw.Dial()")
		}
		return kcp.NewConn(config.RemoteAddr, config.DataShard, config.ParityShard, conn)
	}
	return kcp.DialWithOptions(config.RemoteAddr, config.DataShard, config.ParityShard)
}
