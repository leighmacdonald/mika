package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *MikaService) GetUser(context.Context, *pb.UserID) (*pb.User, error) {
	return nil, nil
}

func (s *MikaService) GetUsers(_ *emptypb.Empty, stream *pb.User) error {
	return nil
}
