package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ConfigServer struct {
	pb.UnimplementedConfigServer
}

func (s *ConfigServer) ConfigUpdate(context.Context, *pb.ConfigUpdateParams) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigUpdate not implemented")
}

func (s *ConfigServer) WhiteListAdd(context.Context, *pb.WhiteListParams) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WhiteListAdd not implemented")
}

func (s *ConfigServer) WhiteListDelete(context.Context, *pb.WhiteListDeleteParams) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WhiteListDelete not implemented")
}

func (s *ConfigServer) WhiteList(context.Context, *emptypb.Empty) (*pb.WhiteListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WhiteList not implemented")
}
