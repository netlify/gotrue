package web

import (
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/rpc/hosts"
	"github.com/netlify/gotrue/rpc/hosts/health"
	"google.golang.org/grpc"
)

type rpcWebServer struct {
	*hosts.RpcHost
}

func NewRpcWebServer(a *api.API, hostAndPort string) *rpcWebServer {
	s := hosts.NewRpcHost(a, "rpc-web", hostAndPort, []hosts.RegisterRpcServer{
		func(s *grpc.Server, srv *hosts.RpcHost) {
			hs := health.NewHealthHost(srv)
			health.RegisterHealthServer(s, hs)
		},
	})
	return &rpcWebServer{s}
}
