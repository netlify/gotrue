package config

//go:generate protoc -I=. --go_out=plugins=grpc:. --go_opt=paths=source_relative config.proto

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	"github.com/netlify/gotrue/rpc/hosts"
	"google.golang.org/grpc/codes"
)

type rpcConfigHost struct {
	*hosts.RpcHost
}

var _ ConfigServer = (*rpcConfigHost)(nil)

func NewConfigHost(h *hosts.RpcHost) *rpcConfigHost {
	return &rpcConfigHost{h}
}

func (r *rpcConfigHost) Settings(_ context.Context,
	_ *SettingsRequest) (*SettingsResponse, error) {
	s := r.API.Settings(nil)
	b, err := json.Marshal(s)
	if err != nil {
		return nil, r.RpcErrorf(codes.Internal, "%w", err)
	}
	var res SettingsResponse
	err = proto.Unmarshal(b, &res)
	if err != nil {
		return nil, r.RpcErrorf(codes.Internal, "%w", err)
	}
	return &res, nil

	return &SettingsResponse{}, nil
}
