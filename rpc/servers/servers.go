package servers

import (
	"fmt"

	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/rpc/servers/rpc"
	"github.com/netlify/gotrue/rpc/servers/web"
	"github.com/sirupsen/logrus"
)

func ListenAndServeRPC(a *api.API, globalConfig  *conf.GlobalConfiguration) {
	go func() {
		addr := fmt.Sprintf("%v:%v", globalConfig.API.Host, globalConfig.API.RpcPort)
		logrus.Infof("GoTrue RPC API started on: %s", addr)
		svr := rpc.NewRpcServer(a, addr)
		svr.ListenAndServe()
	}()

	go func() {
		addr := fmt.Sprintf("%v:%v", globalConfig.API.Host, globalConfig.API.RpcWebPort)
		logrus.Infof("GoTrue RPC Web API started on: %s", addr)
		svr := web.NewRpcWebServer(a, addr)
		// TODO: add JWT server options
		svr.ListenAndServe()
	}()
}
