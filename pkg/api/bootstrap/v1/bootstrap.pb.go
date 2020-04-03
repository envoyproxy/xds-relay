// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.20.1
// 	protoc        v3.11.4
// source: bootstrap/v1/bootstrap.proto

package bootstrapv1

import (
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// [#next-free-field: 4]
type Bootstrap struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// xds-relay server settings.
	Server *Server `protobuf:"bytes,1,opt,name=server,proto3" json:"server,omitempty"`
	// Configuration information about the control plane/upstream server.
	ControlPlaneCluster *Cluster `protobuf:"bytes,2,opt,name=control_plane_cluster,json=controlPlaneCluster,proto3" json:"control_plane_cluster,omitempty"`
	// Logging settings.
	Logging *Logging `protobuf:"bytes,3,opt,name=logging,proto3" json:"logging,omitempty"`
	// Request/response cache settings.
	Cache *Cache `protobuf:"bytes,4,opt,name=cache,proto3" json:"cache,omitempty"`
}

func (x *Bootstrap) Reset() {
	*x = Bootstrap{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Bootstrap) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bootstrap) ProtoMessage() {}

func (x *Bootstrap) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bootstrap.ProtoReflect.Descriptor instead.
func (*Bootstrap) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{0}
}

func (x *Bootstrap) GetServer() *Server {
	if x != nil {
		return x.Server
	}
	return nil
}

func (x *Bootstrap) GetControlPlaneCluster() *Cluster {
	if x != nil {
		return x.ControlPlaneCluster
	}
	return nil
}

func (x *Bootstrap) GetLogging() *Logging {
	if x != nil {
		return x.Logging
	}
	return nil
}

func (x *Bootstrap) GetCache() *Cache {
	if x != nil {
		return x.Cache
	}
	return nil
}

// [#next-free-field: 2]
type Server struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The TCP address that the xds-relay server will listen on.
	Address *SocketAddress `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *Server) Reset() {
	*x = Server{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Server) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Server) ProtoMessage() {}

func (x *Server) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Server.ProtoReflect.Descriptor instead.
func (*Server) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{1}
}

func (x *Server) GetAddress() *SocketAddress {
	if x != nil {
		return x.Address
	}
	return nil
}

// [#next-free-field: 2]
type Cluster struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The address for the cluster.
	Address *SocketAddress `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *Cluster) Reset() {
	*x = Cluster{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cluster) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cluster) ProtoMessage() {}

func (x *Cluster) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cluster.ProtoReflect.Descriptor instead.
func (*Cluster) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{2}
}

func (x *Cluster) GetAddress() *SocketAddress {
	if x != nil {
		return x.Address
	}
	return nil
}

// [#next-free-field: 3]
type Logging struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Filepath where logs are emitted. If no filepath is specified, logs will be written to `/dev/null`.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// The logging level. If no logging level is set, the default is `info`.
	Level string `protobuf:"bytes,2,opt,name=level,proto3" json:"level,omitempty"`
}

func (x *Logging) Reset() {
	*x = Logging{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Logging) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Logging) ProtoMessage() {}

func (x *Logging) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Logging.ProtoReflect.Descriptor instead.
func (*Logging) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{3}
}

func (x *Logging) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *Logging) GetLevel() string {
	if x != nil {
		return x.Level
	}
	return ""
}

// [#next-free-field: 3]
type Cache struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Duration before which a key is evicted from the request/response cache. Zero means no expiration time.
	Ttl *duration.Duration `protobuf:"bytes,1,opt,name=ttl,proto3" json:"ttl,omitempty"`
	// The maximum number of keys allowed in the request/response cache.
	MaxEntries *wrappers.Int32Value `protobuf:"bytes,2,opt,name=max_entries,json=maxEntries,proto3" json:"max_entries,omitempty"`
}

func (x *Cache) Reset() {
	*x = Cache{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cache) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cache) ProtoMessage() {}

func (x *Cache) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cache.ProtoReflect.Descriptor instead.
func (*Cache) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{4}
}

func (x *Cache) GetTtl() *duration.Duration {
	if x != nil {
		return x.Ttl
	}
	return nil
}

func (x *Cache) GetMaxEntries() *wrappers.Int32Value {
	if x != nil {
		return x.MaxEntries
	}
	return nil
}

// [#next-free-field: 4]
type SocketAddress struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The address for this socket. Listeners will bind
	// to the address. An empty address is not allowed.
	Address   string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	PortValue uint32 `protobuf:"varint,2,opt,name=port_value,json=portValue,proto3" json:"port_value,omitempty"`
}

func (x *SocketAddress) Reset() {
	*x = SocketAddress{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SocketAddress) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SocketAddress) ProtoMessage() {}

func (x *SocketAddress) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SocketAddress.ProtoReflect.Descriptor instead.
func (*SocketAddress) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{5}
}

func (x *SocketAddress) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *SocketAddress) GetPortValue() uint32 {
	if x != nil {
		return x.PortValue
	}
	return 0
}

var File_bootstrap_v1_bootstrap_proto protoreflect.FileDescriptor

var file_bootstrap_v1_bootstrap_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2f, 0x76, 0x31, 0x2f, 0x62,
	0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09,
	0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70,
	0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0xd4, 0x01, 0x0a, 0x09, 0x42, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70,
	0x12, 0x29, 0x0a, 0x06, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x52, 0x06, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x46, 0x0a, 0x15, 0x63,
	0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x5f, 0x70, 0x6c, 0x61, 0x6e, 0x65, 0x5f, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x6f, 0x6f,
	0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x13,
	0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x50, 0x6c, 0x61, 0x6e, 0x65, 0x43, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x12, 0x2c, 0x0a, 0x07, 0x6c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70,
	0x2e, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x52, 0x07, 0x6c, 0x6f, 0x67, 0x67, 0x69, 0x6e,
	0x67, 0x12, 0x26, 0x0a, 0x05, 0x63, 0x61, 0x63, 0x68, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x10, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x43, 0x61, 0x63,
	0x68, 0x65, 0x52, 0x05, 0x63, 0x61, 0x63, 0x68, 0x65, 0x22, 0x3c, 0x0a, 0x06, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x12, 0x32, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70,
	0x2e, 0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x52, 0x07,
	0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x3d, 0x0a, 0x07, 0x43, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x12, 0x32, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e,
	0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x52, 0x07, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x33, 0x0a, 0x07, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e,
	0x67, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x22, 0x72, 0x0a, 0x05, 0x43,
	0x61, 0x63, 0x68, 0x65, 0x12, 0x2b, 0x0a, 0x03, 0x74, 0x74, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x03, 0x74, 0x74,
	0x6c, 0x12, 0x3c, 0x0a, 0x0b, 0x6d, 0x61, 0x78, 0x5f, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x0a, 0x6d, 0x61, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x22,
	0x5c, 0x0a, 0x0d, 0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x12, 0x21, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x72, 0x02, 0x20, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x12, 0x28, 0x0a, 0x0a, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x09, 0xfa, 0x42, 0x06, 0x2a, 0x04, 0x18, 0xff,
	0xff, 0x03, 0x52, 0x09, 0x70, 0x6f, 0x72, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x1a, 0x5a,
	0x18, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2f, 0x76, 0x31, 0x3b, 0x62, 0x6f,
	0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_bootstrap_v1_bootstrap_proto_rawDescOnce sync.Once
	file_bootstrap_v1_bootstrap_proto_rawDescData = file_bootstrap_v1_bootstrap_proto_rawDesc
)

func file_bootstrap_v1_bootstrap_proto_rawDescGZIP() []byte {
	file_bootstrap_v1_bootstrap_proto_rawDescOnce.Do(func() {
		file_bootstrap_v1_bootstrap_proto_rawDescData = protoimpl.X.CompressGZIP(file_bootstrap_v1_bootstrap_proto_rawDescData)
	})
	return file_bootstrap_v1_bootstrap_proto_rawDescData
}

var file_bootstrap_v1_bootstrap_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_bootstrap_v1_bootstrap_proto_goTypes = []interface{}{
	(*Bootstrap)(nil),           // 0: bootstrap.Bootstrap
	(*Server)(nil),              // 1: bootstrap.Server
	(*Cluster)(nil),             // 2: bootstrap.Cluster
	(*Logging)(nil),             // 3: bootstrap.Logging
	(*Cache)(nil),               // 4: bootstrap.Cache
	(*SocketAddress)(nil),       // 5: bootstrap.SocketAddress
	(*duration.Duration)(nil),   // 6: google.protobuf.Duration
	(*wrappers.Int32Value)(nil), // 7: google.protobuf.Int32Value
}
var file_bootstrap_v1_bootstrap_proto_depIdxs = []int32{
	1, // 0: bootstrap.Bootstrap.server:type_name -> bootstrap.Server
	2, // 1: bootstrap.Bootstrap.control_plane_cluster:type_name -> bootstrap.Cluster
	3, // 2: bootstrap.Bootstrap.logging:type_name -> bootstrap.Logging
	4, // 3: bootstrap.Bootstrap.cache:type_name -> bootstrap.Cache
	5, // 4: bootstrap.Server.address:type_name -> bootstrap.SocketAddress
	5, // 5: bootstrap.Cluster.address:type_name -> bootstrap.SocketAddress
	6, // 6: bootstrap.Cache.ttl:type_name -> google.protobuf.Duration
	7, // 7: bootstrap.Cache.max_entries:type_name -> google.protobuf.Int32Value
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_bootstrap_v1_bootstrap_proto_init() }
func file_bootstrap_v1_bootstrap_proto_init() {
	if File_bootstrap_v1_bootstrap_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_bootstrap_v1_bootstrap_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Bootstrap); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Server); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cluster); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Logging); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cache); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SocketAddress); i {
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
			RawDescriptor: file_bootstrap_v1_bootstrap_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_bootstrap_v1_bootstrap_proto_goTypes,
		DependencyIndexes: file_bootstrap_v1_bootstrap_proto_depIdxs,
		MessageInfos:      file_bootstrap_v1_bootstrap_proto_msgTypes,
	}.Build()
	File_bootstrap_v1_bootstrap_proto = out.File
	file_bootstrap_v1_bootstrap_proto_rawDesc = nil
	file_bootstrap_v1_bootstrap_proto_goTypes = nil
	file_bootstrap_v1_bootstrap_proto_depIdxs = nil
}
