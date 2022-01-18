// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: cluster/v1/nodeinfo.proto

package clusterv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NodeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Host      string `protobuf:"bytes,2,opt,name=host,proto3" json:"host,omitempty"`
	Hash      uint64 `protobuf:"varint,3,opt,name=hash,proto3" json:"hash,omitempty"`
	Birthdate int64  `protobuf:"varint,4,opt,name=birthdate,proto3" json:"birthdate,omitempty"`
}

func (x *NodeInfo) Reset() {
	*x = NodeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cluster_v1_nodeinfo_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeInfo) ProtoMessage() {}

func (x *NodeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_v1_nodeinfo_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeInfo.ProtoReflect.Descriptor instead.
func (*NodeInfo) Descriptor() ([]byte, []int) {
	return file_cluster_v1_nodeinfo_proto_rawDescGZIP(), []int{0}
}

func (x *NodeInfo) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *NodeInfo) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *NodeInfo) GetHash() uint64 {
	if x != nil {
		return x.Hash
	}
	return 0
}

func (x *NodeInfo) GetBirthdate() int64 {
	if x != nil {
		return x.Birthdate
	}
	return 0
}

var File_cluster_v1_nodeinfo_proto protoreflect.FileDescriptor

var file_cluster_v1_nodeinfo_proto_rawDesc = []byte{
	0x0a, 0x19, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x6e, 0x6f, 0x64,
	0x65, 0x69, 0x6e, 0x66, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x22, 0x60, 0x0a, 0x08, 0x4e, 0x6f, 0x64, 0x65, 0x49,
	0x6e, 0x66, 0x6f, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x1c, 0x0a, 0x09, 0x62,
	0x69, 0x72, 0x74, 0x68, 0x64, 0x61, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09,
	0x62, 0x69, 0x72, 0x74, 0x68, 0x64, 0x61, 0x74, 0x65, 0x42, 0x1c, 0x5a, 0x1a, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x6c,
	0x75, 0x73, 0x74, 0x65, 0x72, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cluster_v1_nodeinfo_proto_rawDescOnce sync.Once
	file_cluster_v1_nodeinfo_proto_rawDescData = file_cluster_v1_nodeinfo_proto_rawDesc
)

func file_cluster_v1_nodeinfo_proto_rawDescGZIP() []byte {
	file_cluster_v1_nodeinfo_proto_rawDescOnce.Do(func() {
		file_cluster_v1_nodeinfo_proto_rawDescData = protoimpl.X.CompressGZIP(file_cluster_v1_nodeinfo_proto_rawDescData)
	})
	return file_cluster_v1_nodeinfo_proto_rawDescData
}

var file_cluster_v1_nodeinfo_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_cluster_v1_nodeinfo_proto_goTypes = []interface{}{
	(*NodeInfo)(nil), // 0: cluster.v1.NodeInfo
}
var file_cluster_v1_nodeinfo_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_cluster_v1_nodeinfo_proto_init() }
func file_cluster_v1_nodeinfo_proto_init() {
	if File_cluster_v1_nodeinfo_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cluster_v1_nodeinfo_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NodeInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cluster_v1_nodeinfo_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cluster_v1_nodeinfo_proto_goTypes,
		DependencyIndexes: file_cluster_v1_nodeinfo_proto_depIdxs,
		MessageInfos:      file_cluster_v1_nodeinfo_proto_msgTypes,
	}.Build()
	File_cluster_v1_nodeinfo_proto = out.File
	file_cluster_v1_nodeinfo_proto_rawDesc = nil
	file_cluster_v1_nodeinfo_proto_goTypes = nil
	file_cluster_v1_nodeinfo_proto_depIdxs = nil
}
