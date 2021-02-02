package rpc

import (
	"context"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
)

func (s *MikaService) RoleAll(_ *emptypb.Empty, stream pb.Mika_RoleAllServer) error {
	log.Debugf("RoleAll request started")
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

func RoleToPB(r *store.Role) *pb.Role {
	return &pb.Role{
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
	}
}

func PBToRole(r *pb.Role) *store.Role {
	return &store.Role{
		RoleID:          r.RoleId,
		RemoteID:        r.RemoteId,
		RoleName:        r.RoleName,
		Priority:        r.Priority,
		MultiUp:         r.MultiUp,
		MultiDown:       r.MultiDown,
		DownloadEnabled: r.DownloadEnabled,
		UploadEnabled:   r.UploadEnabled,
		CreatedOn:       r.Time.CreatedOn.AsTime(),
		UpdateOn:        r.Time.UpdatedOn.AsTime(),
	}
}

func (s *MikaService) RoleAdd(ctx context.Context, params *pb.RoleAddParams) (*pb.Role, error) {
	log.Debugf("RoleAdd request started")
	r := &store.Role{
		RemoteID:        params.RemoteId,
		RoleName:        params.RoleName,
		Priority:        params.Priority,
		MultiUp:         params.MultiUp,
		MultiDown:       params.MultiDown,
		DownloadEnabled: params.UploadEnabled,
		UploadEnabled:   params.UploadEnabled,
	}
	if err := tracker.RoleAdd(r); err != nil {
		return nil, errors.Wrapf(err, "Failed to add role: %s", err.Error())
	}
	return RoleToPB(r), nil
}

func (s *MikaService) RoleDelete(ctx context.Context, roleID *pb.RoleID) (*emptypb.Empty, error) {
	var rID uint32
	if roleID.RoleId > 0 {
		rID = roleID.RoleId
	} else if roleID.RoleName != "" {
		for _, role := range tracker.RoleAll() {
			if strings.ToLower(role.RoleName) == roleID.RoleName {
				rID = role.RoleID
				break
			}
		}
	}
	if rID <= 0 {
		return nil, status.Errorf(codes.NotFound, "role does not exist")
	}
	if err := tracker.RoleDelete(rID); err != nil {
		return nil, errors.Wrapf(err, "Failed to delete role: %s", err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *MikaService) RoleSave(context.Context, *pb.Role) (*emptypb.Empty, error) {
	log.Debugf("RoleSave request started")
	return nil, status.Errorf(codes.Unimplemented, "method RoleSave not implemented")
}
