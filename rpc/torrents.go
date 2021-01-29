package rpc

import (
	pb "github.com/leighmacdonald/mika/proto"
)

type TorrentServer struct {
	pb.UnimplementedTorrentsServer
}
