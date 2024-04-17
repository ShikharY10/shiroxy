// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.12.4
// source: cmd/shiroxy/storage/storage.proto

package storage

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

type DomainMetadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DomainName   string            `protobuf:"bytes,1,opt,name=domain_name,json=domainName,proto3" json:"domain_name,omitempty"`
	CombinedCert []byte            `protobuf:"bytes,2,opt,name=combined_cert,json=combinedCert,proto3" json:"combined_cert,omitempty"`
	Metadata     map[string]string `protobuf:"bytes,3,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *DomainMetadata) Reset() {
	*x = DomainMetadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_shiroxy_storage_storage_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DomainMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DomainMetadata) ProtoMessage() {}

func (x *DomainMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_shiroxy_storage_storage_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DomainMetadata.ProtoReflect.Descriptor instead.
func (*DomainMetadata) Descriptor() ([]byte, []int) {
	return file_cmd_shiroxy_storage_storage_proto_rawDescGZIP(), []int{0}
}

func (x *DomainMetadata) GetDomainName() string {
	if x != nil {
		return x.DomainName
	}
	return ""
}

func (x *DomainMetadata) GetCombinedCert() []byte {
	if x != nil {
		return x.CombinedCert
	}
	return nil
}

func (x *DomainMetadata) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

var File_cmd_shiroxy_storage_storage_proto protoreflect.FileDescriptor

var file_cmd_shiroxy_storage_storage_proto_rawDesc = []byte{
	0x0a, 0x21, 0x63, 0x6d, 0x64, 0x2f, 0x73, 0x68, 0x69, 0x72, 0x6f, 0x78, 0x79, 0x2f, 0x73, 0x74,
	0x6f, 0x72, 0x61, 0x67, 0x65, 0x2f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x22, 0xd3, 0x01, 0x0a, 0x0e, 0x44, 0x6f,
	0x6d, 0x61, 0x69, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1f, 0x0a, 0x0b,
	0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a,
	0x0d, 0x63, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x64, 0x5f, 0x63, 0x65, 0x72, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x63, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x64, 0x43, 0x65,
	0x72, 0x74, 0x12, 0x3e, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x44, 0x6f, 0x6d, 0x61,
	0x69, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x1a, 0x3b, 0x0a, 0x0d, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42,
	0x17, 0x5a, 0x15, 0x2e, 0x2f, 0x63, 0x6d, 0x64, 0x2f, 0x73, 0x68, 0x69, 0x72, 0x6f, 0x78, 0x79,
	0x2f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cmd_shiroxy_storage_storage_proto_rawDescOnce sync.Once
	file_cmd_shiroxy_storage_storage_proto_rawDescData = file_cmd_shiroxy_storage_storage_proto_rawDesc
)

func file_cmd_shiroxy_storage_storage_proto_rawDescGZIP() []byte {
	file_cmd_shiroxy_storage_storage_proto_rawDescOnce.Do(func() {
		file_cmd_shiroxy_storage_storage_proto_rawDescData = protoimpl.X.CompressGZIP(file_cmd_shiroxy_storage_storage_proto_rawDescData)
	})
	return file_cmd_shiroxy_storage_storage_proto_rawDescData
}

var file_cmd_shiroxy_storage_storage_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_cmd_shiroxy_storage_storage_proto_goTypes = []interface{}{
	(*DomainMetadata)(nil), // 0: main.DomainMetadata
	nil,                    // 1: main.DomainMetadata.MetadataEntry
}
var file_cmd_shiroxy_storage_storage_proto_depIdxs = []int32{
	1, // 0: main.DomainMetadata.metadata:type_name -> main.DomainMetadata.MetadataEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_cmd_shiroxy_storage_storage_proto_init() }
func file_cmd_shiroxy_storage_storage_proto_init() {
	if File_cmd_shiroxy_storage_storage_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cmd_shiroxy_storage_storage_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DomainMetadata); i {
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
			RawDescriptor: file_cmd_shiroxy_storage_storage_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cmd_shiroxy_storage_storage_proto_goTypes,
		DependencyIndexes: file_cmd_shiroxy_storage_storage_proto_depIdxs,
		MessageInfos:      file_cmd_shiroxy_storage_storage_proto_msgTypes,
	}.Build()
	File_cmd_shiroxy_storage_storage_proto = out.File
	file_cmd_shiroxy_storage_storage_proto_rawDesc = nil
	file_cmd_shiroxy_storage_storage_proto_goTypes = nil
	file_cmd_shiroxy_storage_storage_proto_depIdxs = nil
}
