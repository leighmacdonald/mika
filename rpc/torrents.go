package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *MikaService) GetTorrent(context.Context, *pb.TorrentParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTorrent not implemented")
}
func (s *MikaService) TorrentAdd(context.Context, *pb.TorrentAddParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentAdd not implemented")
}
func (s *MikaService) TorrentDelete(context.Context, *pb.InfoHashParam) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentDelete not implemented")
}
func (s *MikaService) TorrentUpdate(context.Context, *pb.TorrentUpdateParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentUpdate not implemented")
}
