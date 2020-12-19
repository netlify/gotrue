package rpc

import (
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/rpc/hosts"
	"github.com/netlify/gotrue/rpc/hosts/config"
	"github.com/netlify/gotrue/rpc/hosts/health"
	"google.golang.org/grpc"
)

type rpcServer struct {
	*hosts.RpcHost
}

func NewRpcServer(a *api.API, hostAndPort string) *rpcServer {
	s := hosts.NewRpcHost(a, "rpc", hostAndPort, []hosts.RegisterRpcServer{
		func(s *grpc.Server, srv *hosts.RpcHost) {
			ch := config.NewConfigHost(srv)
			config.RegisterConfigServer(s, ch)
		},
		func(s *grpc.Server, srv *hosts.RpcHost) {
			hh := health.NewHealthHost(srv)
			health.RegisterHealthServer(s, hh)
		},
	})
	return &rpcServer{s}
}
