//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of simple-tls.
//
//     simple-tls is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     simple-tls is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/chunqian/simple-tls/core/grpc_tunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Server struct {
	BindAddr              string
	DstAddr               string
	GRPC                  bool
	GRPCServiceName       string
	Cert, Key, ServerName string
	IdleTimeout           time.Duration
	OutboundBuf           int
	InboundBuf            int

	testListener         net.Listener
	testCert             *tls.Certificate
	testTransportHandler TransportHandler
}

var errMissingCertOrKey = errors.New("one of cert or key argument is missing")

func (s *Server) ActiveAndServe() error {
	var l net.Listener
	if s.testListener != nil {
		l = s.testListener
	} else {
		var err error
		l, err = net.Listen("tcp", s.BindAddr)
		if err != nil {
			return err
		}
	}

	var certificate tls.Certificate
	if s.testCert != nil {
		certificate = *s.testCert
	} else {
		envCert := os.Getenv("SIMPLE_TLS_CERT")
		envKey := os.Getenv("SIMPLE_TLS_KEY")
		switch {
		case len(envCert) > 0 && len(envKey) > 0: // cert and key from env
			cer, err := tls.X509KeyPair([]byte(envCert), []byte(envKey))
			if err != nil {
				return fmt.Errorf("failed load x509 key pair from env: %w", err)
			}
			certificate = cer
		case len(s.Cert) == 0 && len(s.Key) == 0: // no cert and key
			dnsName, _, keyPEM, certPEM, err := GenerateCertificate(s.ServerName, nil)
			if err != nil {
				return fmt.Errorf("failed to generate temp cert: %w", err)
			}

			log.Printf("warnning: you are using a tmp certificate with dns name: %s", dnsName)
			cer, err := tls.X509KeyPair(certPEM, keyPEM)
			if err != nil {
				return fmt.Errorf("cannot load x509 key pair from memory: %w", err)
			}

			certificate = cer
		case len(s.Cert) != 0 && len(s.Key) != 0: // has a cert and a key
			cer, err := tls.LoadX509KeyPair(s.Cert, s.Key) //load cert
			if err != nil {
				return fmt.Errorf("cannot load x509 key pair from disk: %w", err)
			}
			certificate = cer
		default:
			return errMissingCertOrKey
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		VerifyConnection: func(state tls.ConnectionState) error {
			if state.Version != tls.VersionTLS13 {
				return fmt.Errorf("unsafe tls version %d", state.Version)
			}
			return nil
		},
	}

	outboundHandler := func(dst string) TransportHandler {
		var handler TransportHandler
		if s.testTransportHandler != nil {
			handler = s.testTransportHandler
		} else {
			handler = NewDstTransportHandler(dst, s.IdleTimeout, s.OutboundBuf)
		}
		return handler
	}

	if s.GRPC {
		serverOpts := []grpc.ServerOption{
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle: time.Second * 300,
				Time:              time.Second * 60,
				Timeout:           time.Second * 20,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             time.Second * 10,
				PermitWithoutStream: false,
			}),
			grpc.MaxSendMsgSize(64 * 1024),
			grpc.MaxRecvMsgSize(64 * 1024),
			grpc.Creds(credentials.NewTLS(tlsConfig)),
			grpc.InitialWindowSize(1024 * 1024),
			grpc.InitialConnWindowSize(1024 * 1024),
			grpc.MaxConcurrentStreams(64), // This limit is larger than the hardcoded client limit.
			grpc.MaxHeaderListSize(2048),
		}
		grpcServer := grpc.NewServer(serverOpts...)
		if d := s.DstAddr; strings.ContainsAny(d, "/,") {
			pathDstPeers := strings.Split(s.DstAddr, ",")
			for _, peer := range pathDstPeers {
				path, dst, ok := strings.Cut(peer, "/")
				if !ok {
					return fmt.Errorf("invalid dst value [%s]", peer)
				}
				log.Printf("starting grpc func at path %s -> %s", path, dst)
				grpc_tunnel.RegisterGRPCTunnelServerAddon(grpcServer, newGrpcServerHandler(outboundHandler(dst)), path)
			}
		} else {
			grpc_tunnel.RegisterGRPCTunnelServerAddon(grpcServer, newGrpcServerHandler(outboundHandler(s.DstAddr)), s.GRPCServiceName)
		}

		return grpcServer.Serve(l)
	}

	l = tls.NewListener(l, tlsConfig)
	return ListenRawConn(l, outboundHandler(s.DstAddr))
}
