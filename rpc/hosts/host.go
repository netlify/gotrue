package hosts

import (
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
)


/*
var

func RegisterServer(reg func(s *grpc.Server, srv *RpcHost)) {
	servers = append(servers, reg)
}

*/

type RegisterRpcServer func(s *grpc.Server, srv *RpcHost)

type RpcHost struct {
	*api.API
	*logrus.Entry
	hostAndPort string
	servers []RegisterRpcServer
}

func NewRpcHost(a *api.API, name string, hostAndPort string, servers []RegisterRpcServer) *RpcHost {
	log := logrus.WithField("server", name)
	return &RpcHost{a, log,  hostAndPort,servers}
}

func (h *RpcHost) ListenAndServe(opts ...grpc.ServerOption) {
	lis, err := net.Listen("tcp", h.hostAndPort)
	if err != nil {
		h.WithError(err).Fatal("rpc server listen failed")
	}

	server := grpc.NewServer(opts...)
	for _, s := range h.servers {
		s(server, h)
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		util.WaitForTermination(h, done)
		h.Info("shutting down rpc server...")
		server.GracefulStop()
	}()

	if err := server.Serve(lis); err != nil {
		h.WithError(err).Fatal("rpc server failed to start")
	}
}

func (h *RpcHost) RpcErrorf(c codes.Code, format string, a ...interface{}) error {
	err := status.Errorf(c, format, a...)
	h.Error(err)
	return err
}

func (h *RpcHost) RpcError(c codes.Code, msg string) error {
	err := status.Error(c, msg)
	h.Error(err)
	return err
}

