// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: proto/mika.proto

package rpc

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

var File_proto_mika_proto protoreflect.FileDescriptor

var file_proto_mika_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x04, 0x6d, 0x69, 0x6b, 0x61, 0x1a, 0x12, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x13, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x6f, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x32, 0xe6, 0x07, 0x0a, 0x04, 0x4d, 0x69, 0x6b, 0x61, 0x12, 0x42, 0x0a, 0x0c, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x18, 0x2e, 0x6d, 0x69,
	0x6b, 0x61, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50,
	0x61, 0x72, 0x61, 0x6d, 0x73, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12,
	0x39, 0x0a, 0x0c, 0x57, 0x68, 0x69, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x64, 0x64, 0x12,
	0x0f, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x57, 0x68, 0x69, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x48, 0x0a, 0x0f, 0x57, 0x68,
	0x69, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x1b, 0x2e,
	0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x57, 0x68, 0x69, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x12, 0x3b, 0x0a, 0x0c, 0x57, 0x68, 0x69, 0x74, 0x65, 0x4c, 0x69, 0x73,
	0x74, 0x41, 0x6c, 0x6c, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0f, 0x2e, 0x6d,
	0x69, 0x6b, 0x61, 0x2e, 0x57, 0x68, 0x69, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x00, 0x30,
	0x01, 0x12, 0x37, 0x0a, 0x0a, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x41, 0x6c, 0x6c, 0x12,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0d, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x54,
	0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x22, 0x00, 0x30, 0x01, 0x12, 0x32, 0x0a, 0x0a, 0x54, 0x6f,
	0x72, 0x72, 0x65, 0x6e, 0x74, 0x47, 0x65, 0x74, 0x12, 0x13, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e,
	0x49, 0x6e, 0x66, 0x6f, 0x48, 0x61, 0x73, 0x68, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x1a, 0x0d, 0x2e,
	0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x22, 0x00, 0x12, 0x35,
	0x0a, 0x0a, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x41, 0x64, 0x64, 0x12, 0x16, 0x2e, 0x6d,
	0x69, 0x6b, 0x61, 0x2e, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x41, 0x64, 0x64, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x73, 0x1a, 0x0d, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x54, 0x6f, 0x72, 0x72,
	0x65, 0x6e, 0x74, 0x22, 0x00, 0x12, 0x3e, 0x0a, 0x0d, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x13, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x49, 0x6e,
	0x66, 0x6f, 0x48, 0x61, 0x73, 0x68, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x1a, 0x16, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x3b, 0x0a, 0x0d, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x19, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x54, 0x6f,
	0x72, 0x72, 0x65, 0x6e, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x73, 0x1a, 0x0d, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x22, 0x00, 0x12, 0x25, 0x0a, 0x07, 0x55, 0x73, 0x65, 0x72, 0x47, 0x65, 0x74, 0x12, 0x0c, 0x2e,
	0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x49, 0x44, 0x1a, 0x0a, 0x2e, 0x6d, 0x69,
	0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x22, 0x00, 0x12, 0x31, 0x0a, 0x07, 0x55, 0x73, 0x65,
	0x72, 0x41, 0x6c, 0x6c, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0a, 0x2e, 0x6d,
	0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x22, 0x00, 0x30, 0x01, 0x12, 0x30, 0x0a, 0x08,
	0x55, 0x73, 0x65, 0x72, 0x53, 0x61, 0x76, 0x65, 0x12, 0x16, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e,
	0x55, 0x73, 0x65, 0x72, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73,
	0x1a, 0x0a, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x22, 0x00, 0x12, 0x34,
	0x0a, 0x0a, 0x55, 0x73, 0x65, 0x72, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x0c, 0x2e, 0x6d,
	0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x49, 0x44, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x12, 0x2c, 0x0a, 0x07, 0x55, 0x73, 0x65, 0x72, 0x41, 0x64, 0x64, 0x12,
	0x13, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x41, 0x64, 0x64, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x73, 0x1a, 0x0a, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x55, 0x73, 0x65, 0x72,
	0x22, 0x00, 0x12, 0x31, 0x0a, 0x07, 0x52, 0x6f, 0x6c, 0x65, 0x41, 0x6c, 0x6c, 0x12, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0a, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x52, 0x6f, 0x6c,
	0x65, 0x22, 0x00, 0x30, 0x01, 0x12, 0x2c, 0x0a, 0x07, 0x52, 0x6f, 0x6c, 0x65, 0x41, 0x64, 0x64,
	0x12, 0x13, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x52, 0x6f, 0x6c, 0x65, 0x41, 0x64, 0x64, 0x50,
	0x61, 0x72, 0x61, 0x6d, 0x73, 0x1a, 0x0a, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x52, 0x6f, 0x6c,
	0x65, 0x22, 0x00, 0x12, 0x34, 0x0a, 0x0a, 0x52, 0x6f, 0x6c, 0x65, 0x44, 0x65, 0x6c, 0x65, 0x74,
	0x65, 0x12, 0x0c, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x52, 0x6f, 0x6c, 0x65, 0x49, 0x44, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x30, 0x0a, 0x08, 0x52, 0x6f, 0x6c,
	0x65, 0x53, 0x61, 0x76, 0x65, 0x12, 0x0a, 0x2e, 0x6d, 0x69, 0x6b, 0x61, 0x2e, 0x52, 0x6f, 0x6c,
	0x65, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x42, 0x24, 0x5a, 0x22, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x65, 0x69, 0x67, 0x68, 0x6d,
	0x61, 0x63, 0x64, 0x6f, 0x6e, 0x61, 0x6c, 0x64, 0x2f, 0x6d, 0x69, 0x6b, 0x61, 0x2f, 0x72, 0x70,
	0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_proto_mika_proto_goTypes = []interface{}{
	(*ConfigUpdateParams)(nil),    // 0: mika.ConfigUpdateParams
	(*WhiteList)(nil),             // 1: mika.WhiteList
	(*WhiteListDeleteParams)(nil), // 2: mika.WhiteListDeleteParams
	(*emptypb.Empty)(nil),         // 3: google.protobuf.Empty
	(*InfoHashParam)(nil),         // 4: mika.InfoHashParam
	(*TorrentAddParams)(nil),      // 5: mika.TorrentAddParams
	(*TorrentUpdateParams)(nil),   // 6: mika.TorrentUpdateParams
	(*UserID)(nil),                // 7: mika.UserID
	(*UserUpdateParams)(nil),      // 8: mika.UserUpdateParams
	(*UserAddParams)(nil),         // 9: mika.UserAddParams
	(*RoleAddParams)(nil),         // 10: mika.RoleAddParams
	(*RoleID)(nil),                // 11: mika.RoleID
	(*Role)(nil),                  // 12: mika.Role
	(*Torrent)(nil),               // 13: mika.Torrent
	(*User)(nil),                  // 14: mika.User
}
var file_proto_mika_proto_depIdxs = []int32{
	0,  // 0: mika.Mika.ConfigUpdate:input_type -> mika.ConfigUpdateParams
	1,  // 1: mika.Mika.WhiteListAdd:input_type -> mika.WhiteList
	2,  // 2: mika.Mika.WhiteListDelete:input_type -> mika.WhiteListDeleteParams
	3,  // 3: mika.Mika.WhiteListAll:input_type -> google.protobuf.Empty
	3,  // 4: mika.Mika.TorrentAll:input_type -> google.protobuf.Empty
	4,  // 5: mika.Mika.TorrentGet:input_type -> mika.InfoHashParam
	5,  // 6: mika.Mika.TorrentAdd:input_type -> mika.TorrentAddParams
	4,  // 7: mika.Mika.TorrentDelete:input_type -> mika.InfoHashParam
	6,  // 8: mika.Mika.TorrentUpdate:input_type -> mika.TorrentUpdateParams
	7,  // 9: mika.Mika.UserGet:input_type -> mika.UserID
	3,  // 10: mika.Mika.UserAll:input_type -> google.protobuf.Empty
	8,  // 11: mika.Mika.UserSave:input_type -> mika.UserUpdateParams
	7,  // 12: mika.Mika.UserDelete:input_type -> mika.UserID
	9,  // 13: mika.Mika.UserAdd:input_type -> mika.UserAddParams
	3,  // 14: mika.Mika.RoleAll:input_type -> google.protobuf.Empty
	10, // 15: mika.Mika.RoleAdd:input_type -> mika.RoleAddParams
	11, // 16: mika.Mika.RoleDelete:input_type -> mika.RoleID
	12, // 17: mika.Mika.RoleSave:input_type -> mika.Role
	3,  // 18: mika.Mika.ConfigUpdate:output_type -> google.protobuf.Empty
	3,  // 19: mika.Mika.WhiteListAdd:output_type -> google.protobuf.Empty
	3,  // 20: mika.Mika.WhiteListDelete:output_type -> google.protobuf.Empty
	1,  // 21: mika.Mika.WhiteListAll:output_type -> mika.WhiteList
	13, // 22: mika.Mika.TorrentAll:output_type -> mika.Torrent
	13, // 23: mika.Mika.TorrentGet:output_type -> mika.Torrent
	13, // 24: mika.Mika.TorrentAdd:output_type -> mika.Torrent
	3,  // 25: mika.Mika.TorrentDelete:output_type -> google.protobuf.Empty
	13, // 26: mika.Mika.TorrentUpdate:output_type -> mika.Torrent
	14, // 27: mika.Mika.UserGet:output_type -> mika.User
	14, // 28: mika.Mika.UserAll:output_type -> mika.User
	14, // 29: mika.Mika.UserSave:output_type -> mika.User
	3,  // 30: mika.Mika.UserDelete:output_type -> google.protobuf.Empty
	14, // 31: mika.Mika.UserAdd:output_type -> mika.User
	12, // 32: mika.Mika.RoleAll:output_type -> mika.Role
	12, // 33: mika.Mika.RoleAdd:output_type -> mika.Role
	3,  // 34: mika.Mika.RoleDelete:output_type -> google.protobuf.Empty
	3,  // 35: mika.Mika.RoleSave:output_type -> google.protobuf.Empty
	18, // [18:36] is the sub-list for method output_type
	0,  // [0:18] is the sub-list for method input_type
	0,  // [0:0] is the sub-list for extension type_name
	0,  // [0:0] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_proto_mika_proto_init() }
func file_proto_mika_proto_init() {
	if File_proto_mika_proto != nil {
		return
	}
	file_proto_config_proto_init()
	file_proto_tracker_proto_init()
	file_proto_role_proto_init()
	file_proto_user_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_mika_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_mika_proto_goTypes,
		DependencyIndexes: file_proto_mika_proto_depIdxs,
	}.Build()
	File_proto_mika_proto = out.File
	file_proto_mika_proto_rawDesc = nil
	file_proto_mika_proto_goTypes = nil
	file_proto_mika_proto_depIdxs = nil
}
