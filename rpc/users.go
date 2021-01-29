package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserServer struct {
	pb.UnimplementedUsersServer
}

func (s *UserServer) GetUser(context.Context, *pb.UserID) (*pb.User, error) {
	return nil, nil
}

func (s *UserServer) GetUsers(_ *emptypb.Empty, stream pb.UsersServer) error {
	return nil
}
