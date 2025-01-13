// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v5.29.3
// source: cluster_status.proto

// Package status contains generated code for reading and writing the ClusterStatus protobuf.
package status

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

type StatusType int32

const (
	StatusType_STATUS_TYPE_UNSPECIFIED  StatusType = 0
	StatusType_STATUS_TYPE_INIT_STARTED StatusType = 1
	StatusType_STATUS_TYPE_INIT_OK      StatusType = 2
	StatusType_STATUS_TYPE_INIT_FAILED  StatusType = 3
	StatusType_STATUS_TYPE_POD_STARTED  StatusType = 4
	StatusType_STATUS_TYPE_POD_STOPPING StatusType = 5
)

// Enum value maps for StatusType.
var (
	StatusType_name = map[int32]string{
		0: "STATUS_TYPE_UNSPECIFIED",
		1: "STATUS_TYPE_INIT_STARTED",
		2: "STATUS_TYPE_INIT_OK",
		3: "STATUS_TYPE_INIT_FAILED",
		4: "STATUS_TYPE_POD_STARTED",
		5: "STATUS_TYPE_POD_STOPPING",
	}
	StatusType_value = map[string]int32{
		"STATUS_TYPE_UNSPECIFIED":  0,
		"STATUS_TYPE_INIT_STARTED": 1,
		"STATUS_TYPE_INIT_OK":      2,
		"STATUS_TYPE_INIT_FAILED":  3,
		"STATUS_TYPE_POD_STARTED":  4,
		"STATUS_TYPE_POD_STOPPING": 5,
	}
)

func (x StatusType) Enum() *StatusType {
	p := new(StatusType)
	*p = x
	return p
}

func (x StatusType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StatusType) Descriptor() protoreflect.EnumDescriptor {
	return file_cluster_status_proto_enumTypes[0].Descriptor()
}

func (StatusType) Type() protoreflect.EnumType {
	return &file_cluster_status_proto_enumTypes[0]
}

func (x StatusType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StatusType.Descriptor instead.
func (StatusType) EnumDescriptor() ([]byte, []int) {
	return file_cluster_status_proto_rawDescGZIP(), []int{0}
}

type StatusCheck struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Passing       bool                   `protobuf:"varint,2,opt,name=passing,proto3" json:"passing,omitempty"`
	Error         string                 `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StatusCheck) Reset() {
	*x = StatusCheck{}
	mi := &file_cluster_status_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusCheck) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusCheck) ProtoMessage() {}

func (x *StatusCheck) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_status_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusCheck.ProtoReflect.Descriptor instead.
func (*StatusCheck) Descriptor() ([]byte, []int) {
	return file_cluster_status_proto_rawDescGZIP(), []int{0}
}

func (x *StatusCheck) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *StatusCheck) GetPassing() bool {
	if x != nil {
		return x.Passing
	}
	return false
}

func (x *StatusCheck) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type ClusterStatus struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	Account          string                 `protobuf:"bytes,1,opt,name=account,proto3" json:"account,omitempty"`
	Region           string                 `protobuf:"bytes,2,opt,name=region,proto3" json:"region,omitempty"`
	Name             string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	State            StatusType             `protobuf:"varint,4,opt,name=state,proto3,enum=status.StatusType" json:"state,omitempty"`
	ChartVersion     string                 `protobuf:"bytes,5,opt,name=chart_version,json=chartVersion,proto3" json:"chart_version,omitempty"`
	AgentVersion     string                 `protobuf:"bytes,6,opt,name=agent_version,json=agentVersion,proto3" json:"agent_version,omitempty"`
	ScrapeConfig     string                 `protobuf:"bytes,7,opt,name=scrape_config,json=scrapeConfig,proto3" json:"scrape_config,omitempty"`
	ValidatorVersion string                 `protobuf:"bytes,8,opt,name=validator_version,json=validatorVersion,proto3" json:"validator_version,omitempty"`
	K8SVersion       string                 `protobuf:"bytes,9,opt,name=k8s_version,json=k8sVersion,proto3" json:"k8s_version,omitempty"`
	Checks           []*StatusCheck         `protobuf:"bytes,10,rep,name=checks,proto3" json:"checks,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ClusterStatus) Reset() {
	*x = ClusterStatus{}
	mi := &file_cluster_status_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClusterStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterStatus) ProtoMessage() {}

func (x *ClusterStatus) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_status_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterStatus.ProtoReflect.Descriptor instead.
func (*ClusterStatus) Descriptor() ([]byte, []int) {
	return file_cluster_status_proto_rawDescGZIP(), []int{1}
}

func (x *ClusterStatus) GetAccount() string {
	if x != nil {
		return x.Account
	}
	return ""
}

func (x *ClusterStatus) GetRegion() string {
	if x != nil {
		return x.Region
	}
	return ""
}

func (x *ClusterStatus) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ClusterStatus) GetState() StatusType {
	if x != nil {
		return x.State
	}
	return StatusType_STATUS_TYPE_UNSPECIFIED
}

func (x *ClusterStatus) GetChartVersion() string {
	if x != nil {
		return x.ChartVersion
	}
	return ""
}

func (x *ClusterStatus) GetAgentVersion() string {
	if x != nil {
		return x.AgentVersion
	}
	return ""
}

func (x *ClusterStatus) GetScrapeConfig() string {
	if x != nil {
		return x.ScrapeConfig
	}
	return ""
}

func (x *ClusterStatus) GetValidatorVersion() string {
	if x != nil {
		return x.ValidatorVersion
	}
	return ""
}

func (x *ClusterStatus) GetK8SVersion() string {
	if x != nil {
		return x.K8SVersion
	}
	return ""
}

func (x *ClusterStatus) GetChecks() []*StatusCheck {
	if x != nil {
		return x.Checks
	}
	return nil
}

var File_cluster_status_proto protoreflect.FileDescriptor

var file_cluster_status_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x51,
	0x0a, 0x0b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x73, 0x73, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x70, 0x61, 0x73, 0x73, 0x69, 0x6e, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x22, 0xe9, 0x02, 0x0a, 0x0d, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72,
	0x65, 0x67, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x28, 0x0a, 0x05, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x54, 0x79, 0x70, 0x65, 0x52, 0x05, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x63, 0x68, 0x61, 0x72, 0x74, 0x5f, 0x76, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x63, 0x68, 0x61, 0x72,
	0x74, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a, 0x0d, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0c, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x23, 0x0a,
	0x0d, 0x73, 0x63, 0x72, 0x61, 0x70, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x63, 0x72, 0x61, 0x70, 0x65, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x12, 0x2b, 0x0a, 0x11, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x5f,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12,
	0x1f, 0x0a, 0x0b, 0x6b, 0x38, 0x73, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x09,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x6b, 0x38, 0x73, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x12, 0x2b, 0x0a, 0x06, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x13, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x43, 0x68, 0x65, 0x63, 0x6b, 0x52, 0x06, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x2a, 0xb8, 0x01,
	0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1b, 0x0a, 0x17,
	0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50,
	0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x1c, 0x0a, 0x18, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x49, 0x4e, 0x49, 0x54, 0x5f, 0x53, 0x54,
	0x41, 0x52, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12, 0x17, 0x0a, 0x13, 0x53, 0x54, 0x41, 0x54, 0x55,
	0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x49, 0x4e, 0x49, 0x54, 0x5f, 0x4f, 0x4b, 0x10, 0x02,
	0x12, 0x1b, 0x0a, 0x17, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f,
	0x49, 0x4e, 0x49, 0x54, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x10, 0x03, 0x12, 0x1b, 0x0a,
	0x17, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x50, 0x4f, 0x44,
	0x5f, 0x53, 0x54, 0x41, 0x52, 0x54, 0x45, 0x44, 0x10, 0x04, 0x12, 0x1c, 0x0a, 0x18, 0x53, 0x54,
	0x41, 0x54, 0x55, 0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x50, 0x4f, 0x44, 0x5f, 0x53, 0x54,
	0x4f, 0x50, 0x50, 0x49, 0x4e, 0x47, 0x10, 0x05, 0x42, 0x0b, 0x5a, 0x09, 0x2e, 0x2e, 0x2f, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cluster_status_proto_rawDescOnce sync.Once
	file_cluster_status_proto_rawDescData = file_cluster_status_proto_rawDesc
)

func file_cluster_status_proto_rawDescGZIP() []byte {
	file_cluster_status_proto_rawDescOnce.Do(func() {
		file_cluster_status_proto_rawDescData = protoimpl.X.CompressGZIP(file_cluster_status_proto_rawDescData)
	})
	return file_cluster_status_proto_rawDescData
}

var file_cluster_status_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_cluster_status_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_cluster_status_proto_goTypes = []any{
	(StatusType)(0),       // 0: status.StatusType
	(*StatusCheck)(nil),   // 1: status.StatusCheck
	(*ClusterStatus)(nil), // 2: status.ClusterStatus
}
var file_cluster_status_proto_depIdxs = []int32{
	0, // 0: status.ClusterStatus.state:type_name -> status.StatusType
	1, // 1: status.ClusterStatus.checks:type_name -> status.StatusCheck
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_cluster_status_proto_init() }
func file_cluster_status_proto_init() {
	if File_cluster_status_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cluster_status_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cluster_status_proto_goTypes,
		DependencyIndexes: file_cluster_status_proto_depIdxs,
		EnumInfos:         file_cluster_status_proto_enumTypes,
		MessageInfos:      file_cluster_status_proto_msgTypes,
	}.Build()
	File_cluster_status_proto = out.File
	file_cluster_status_proto_rawDesc = nil
	file_cluster_status_proto_goTypes = nil
	file_cluster_status_proto_depIdxs = nil
}
