package rpc

import (
	"context"
	"github.com/leighmacdonald/mika/consts"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TorrentToPB(r *store.Torrent) *pb.Torrent {
	return &pb.Torrent{
		InfoHash:   r.InfoHash.Bytes(),
		Snatches:   r.Snatches,
		Uploaded:   r.Uploaded,
		Downloaded: r.Downloaded,
		IsDeleted:  r.IsDeleted,
		IsEnabled:  r.IsEnabled,
		Reason:     r.Reason,
		MultiUp:    r.MultiUp,
		MultiDn:    r.MultiDn,
		Announces:  r.Announces,
		Seeders:    r.Seeders,
		Leechers:   r.Leechers,
		Title:      r.Title,
		Time: &pb.TimeMeta{
			CreatedOn: timestamppb.New(r.CreatedOn),
			UpdatedOn: timestamppb.New(r.UpdatedOn),
		},
	}
}

func PBtoTorrent(p *pb.Torrent) *store.Torrent {
	var ih store.InfoHash
	_ = store.InfoHashFromBytes(&ih, p.InfoHash)
	return &store.Torrent{
		InfoHash:   ih,
		Snatches:   p.Snatches,
		Uploaded:   p.Uploaded,
		Downloaded: p.Downloaded,
		IsDeleted:  p.IsDeleted,
		IsEnabled:  p.IsEnabled,
		Reason:     p.Reason,
		MultiUp:    p.MultiUp,
		MultiDn:    p.MultiDn,
		Announces:  p.Announces,
		Seeders:    p.Seeders,
		Leechers:   p.Leechers,
		Title:      p.Title,
		CreatedOn:  p.Time.CreatedOn.AsTime(),
		UpdatedOn:  p.Time.UpdatedOn.AsTime(),
		Peers:      nil,
	}
}

func (s *MikaService) TorrentGet(_ context.Context, params *pb.InfoHashParam) (*pb.Torrent, error) {
	var ih store.InfoHash
	err := store.InfoHashFromBytes(&ih, params.InfoHash)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid info_hash")
	}
	t, err2 := tracker.TorrentGet(ih, false)
	if err2 != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid torrent")
	}
	return TorrentToPB(t), nil
}

func (s *MikaService) TorrentAdd(_ context.Context, params *pb.TorrentAddParams) (*pb.Torrent, error) {
	var ih store.InfoHash
	err := store.InfoHashFromBytes(&ih, params.InfoHash)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid info_hash")
	}
	t := &store.Torrent{
		InfoHash:  ih,
		MultiUp:   params.MultiUp,
		MultiDn:   params.MultiDn,
		Title:     params.Title,
		IsEnabled: true,
	}
	err = tracker.TorrentAdd(t)
	if err != nil {
		if errors.Is(err, consts.ErrDuplicate) {
			return nil, status.Errorf(codes.AlreadyExists, "info_hash already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to add torrent")
	}
	return TorrentToPB(t), nil
}

func (s *MikaService) TorrentDelete(_ context.Context, params *pb.InfoHashParam) (*emptypb.Empty, error) {
	var (
		ih store.InfoHash
		t  *store.Torrent
	)
	err := store.InfoHashFromBytes(&ih, params.InfoHash)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.InvalidArgument, "invalid infohash")
	}
	t, err = tracker.TorrentGet(ih, false)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.NotFound, "unknown infohash")
	}
	if err := tracker.TorrentDelete(t); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete torrent")
	}
	return &emptypb.Empty{}, nil
}

func (s *MikaService) TorrentUpdate(context.Context, *pb.TorrentUpdateParams) (*pb.Torrent, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TorrentSave not implemented")
}

func (s *MikaService) TorrentAll(_ *emptypb.Empty, stream pb.Mika_TorrentAllServer) error {
	var err error
	for _, t := range tracker.Torrents() {
		err = stream.Send(TorrentToPB(t))
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send torrent list")
		}
	}
	return nil
}
