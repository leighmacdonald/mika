package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TorrentServer struct {
	pb.UnimplementedTorrentsServer
}

func (TorrentServer) GetTorrent(context.Context, *pb.TorrentParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTorrent not implemented")
}
func (TorrentServer) TorrentAdd(context.Context, *pb.TorrentAddParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentAdd not implemented")
}
func (TorrentServer) TorrentDelete(context.Context, *pb.InfoHashParam) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentDelete not implemented")
}
func (TorrentServer) TorrentUpdate(context.Context, *pb.TorrentUpdateParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentUpdate not implemented")
}
