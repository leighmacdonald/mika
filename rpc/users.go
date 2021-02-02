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

func findUser(userID *pb.UserID) (*store.User, error) {
	var (
		u   *store.User
		err error
	)
	if userID.UserId > 0 {
		u, err = tracker.UserGetByUserID(userID.UserId)
	} else if userID.Passkey != "" {
		u, err = tracker.UserGetByPasskey(userID.Passkey)
	} else if userID.RemoteId > 0 {
		return nil, err
	} else {
		err = errors.New("must supply at least one identifier")
	}
	return u, err
}

func (s *MikaService) UserGet(_ context.Context, userID *pb.UserID) (*pb.User, error) {
	u, err := findUser(userID)
	if err != nil {
		if errors.Is(err, consts.ErrInvalidUser) {
			return nil, status.Errorf(codes.NotFound, "user doesnt exist")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	return UserToPB(u), err
}

func (s *MikaService) UserAll(_ *emptypb.Empty, stream pb.Mika_UserAllServer) error {
	for _, usr := range tracker.Users() {
		if err := stream.Send(UserToPB(usr)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MikaService) UserSave(_ context.Context, params *pb.UserUpdateParams) (*pb.User, error) {
	usr, err := tracker.UserGetByUserID(params.UserId)
	if err != nil {
		if errors.Is(err, consts.ErrInvalidUser) {
			return nil, status.Errorf(codes.NotFound, "user doesnt exist")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	usr.RoleID = params.RoleId
	usr.RemoteID = params.RemoteId
	usr.UserName = params.UserName
	usr.DownloadEnabled = params.DownloadEnabled
	usr.Downloaded = params.Downloaded
	usr.Uploaded = params.Uploaded
	usr.Passkey = params.Passkey
	if err := tracker.UserSave(usr); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user")
	}
	return UserToPB(usr), nil
}

func (s *MikaService) UserDelete(_ context.Context, userID *pb.UserID) (*emptypb.Empty, error) {
	u, err := findUser(userID)
	if err != nil {
		if errors.Is(err, consts.ErrInvalidUser) {
			return nil, status.Errorf(codes.NotFound, "user doesnt exist")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete user")
	}
	if err := tracker.UserDelete(u); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user")
	}
	return nil, status.Errorf(codes.Unimplemented, "method UserDelete not implemented")
}

func (s *MikaService) UserAdd(ctx context.Context, p *pb.UserAddParams) (*pb.User, error) {
	u := &store.User{
		RoleID:          p.RoleId,
		UserName:        p.UserName,
		Passkey:         p.Passkey,
		IsDeleted:       false,
		DownloadEnabled: p.DownloadEnabled,
		Downloaded:      p.Downloaded,
		Uploaded:        p.Uploaded,
		RemoteID:        p.RemoteId,
	}
	if err := tracker.UserAdd(u); err != nil {
		return nil, err
	}
	return UserToPB(u), nil
}

func UserToPB(u *store.User) *pb.User {
	return &pb.User{
		UserId:     u.UserID,
		RoleId:     u.RoleID,
		RemoteId:   u.RemoteID,
		UserName:   u.UserName,
		Downloaded: u.Downloaded,
		Uploaded:   u.Uploaded,
		Passkey:    u.Passkey,
		Time: &pb.TimeMeta{
			CreatedOn: timestamppb.New(u.CreatedOn),
			UpdatedOn: timestamppb.New(u.UpdatedOn),
		},
		Role: RoleToPB(u.Role),
	}
}

func PBToUser(u *pb.User) *store.User {
	return &store.User{
		UserID:          u.UserId,
		RoleID:          u.RoleId,
		UserName:        u.UserName,
		Passkey:         u.Passkey,
		IsDeleted:       u.IsDeleted,
		DownloadEnabled: u.DownloadEnabled,
		Downloaded:      u.Downloaded,
		Uploaded:        u.Uploaded,
		Announces:       u.Announces,
		RemoteID:        u.RemoteId,
		CreatedOn:       u.Time.CreatedOn.AsTime(),
		UpdatedOn:       u.Time.UpdatedOn.AsTime(),
		Role:            PBToRole(u.Role),
	}
}
