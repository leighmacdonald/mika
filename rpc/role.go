package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/tracker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *MikaService) RoleAll(_ *emptypb.Empty, stream pb.Mika_RoleAllServer) error {
	var err error
	for _, r := range tracker.RoleAll() {
		err = stream.Send(&pb.Role{
			RoleId:          r.RoleID,
			RoleName:        r.RoleName,
			RemoteId:        r.RemoteID,
			Priority:        r.Priority,
			DownloadEnabled: r.DownloadEnabled,
			UploadEnabled:   r.UploadEnabled,
			MultiUp:         r.MultiUp,
			MultiDown:       r.MultiDown,
			Time: &pb.TimeMeta{
				CreatedOn: timestamppb.New(r.CreatedOn),
				UpdatedOn: timestamppb.New(r.UpdateOn),
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to send role")
		}
	}
	return nil
}

func (s *MikaService) RoleAdd(context.Context, *pb.RoleAddParams) (*pb.Role, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RoleAdd not implemented")
}
func (s *MikaService) RoleDelete(context.Context, *pb.RoleID) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RoleDelete not implemented")
}
func (s *MikaService) RoleSave(context.Context, *pb.Role) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RoleSave not implemented")
}
