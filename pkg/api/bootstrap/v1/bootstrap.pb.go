// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.11.4
// source: bootstrap/v1/bootstrap.proto

package bootstrapv1

import (
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
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

// The logging level. If no logging level is set, the default is INFO.
type Logging_Level int32

const (
	Logging_INFO  Logging_Level = 0
	Logging_DEBUG Logging_Level = 1
	Logging_WARN  Logging_Level = 2
	Logging_ERROR Logging_Level = 3
)

// Enum value maps for Logging_Level.
var (
	Logging_Level_name = map[int32]string{
		0: "INFO",
		1: "DEBUG",
		2: "WARN",
		3: "ERROR",
	}
	Logging_Level_value = map[string]int32{
		"INFO":  0,
		"DEBUG": 1,
		"WARN":  2,
		"ERROR": 3,
	}
)

func (x Logging_Level) Enum() *Logging_Level {
	p := new(Logging_Level)
	*p = x
	return p
}

func (x Logging_Level) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Logging_Level) Descriptor() protoreflect.EnumDescriptor {
	return file_bootstrap_v1_bootstrap_proto_enumTypes[0].Descriptor()
}

func (Logging_Level) Type() protoreflect.EnumType {
	return &file_bootstrap_v1_bootstrap_proto_enumTypes[0]
}

func (x Logging_Level) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Logging_Level.Descriptor instead.
func (Logging_Level) EnumDescriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{3, 0}
}

// [#next-free-field: 7]
type Bootstrap struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// xds-relay server configuration.
	Server *Server `protobuf:"bytes,1,opt,name=server,proto3" json:"server,omitempty"`
	// Configuration information about the origin server.
	OriginServer *Upstream `protobuf:"bytes,2,opt,name=origin_server,json=originServer,proto3" json:"origin_server,omitempty"`
	// Logging settings.
	Logging *Logging `protobuf:"bytes,3,opt,name=logging,proto3" json:"logging,omitempty"`
	// Request/response cache settings.
	Cache *Cache `protobuf:"bytes,4,opt,name=cache,proto3" json:"cache,omitempty"`
	// Metrics sink settings
	MetricsSink *MetricsSink `protobuf:"bytes,5,opt,name=metrics_sink,json=metricsSink,proto3" json:"metrics_sink,omitempty"`
	// Admin server configuration.
	Admin *Admin `protobuf:"bytes,6,opt,name=admin,proto3" json:"admin,omitempty"`
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

func (x *Bootstrap) GetOriginServer() *Upstream {
	if x != nil {
		return x.OriginServer
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

func (x *Bootstrap) GetMetricsSink() *MetricsSink {
	if x != nil {
		return x.MetricsSink
	}
	return nil
}

func (x *Bootstrap) GetAdmin() *Admin {
	if x != nil {
		return x.Admin
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

// [#next-free-field: 5]
type Upstream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The address for the upstream cluster.
	Address *SocketAddress `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	// grpc connection keep alive time backed by https://github.com/grpc/grpc-go/blob/v1.32.0/keepalive/keepalive.go#L34-L37
	// If unset defaults to 5 minutes.
	// Usage example: 2m to represent 2 minutes
	// Reason for not using google.protobuf.Duration
	// keepalive will be in minutes or possibly hours.
	// From https://developers.google.com/protocol-buffers/docs/reference/java/com/google/protobuf/Duration
	// Duration only lets us represent time in 's'
	KeepAliveTime string `protobuf:"bytes,2,opt,name=keep_alive_time,json=keepAliveTime,proto3" json:"keep_alive_time,omitempty"`
	// Timeout for upstream connection stream.
	// If unset defaults to no timeout.
	// A new stream will be opened (retried) after the timeout is hit.
	// Usage example: 2m to represent 2 minutes.
	StreamTimeout string `protobuf:"bytes,3,opt,name=stream_timeout,json=streamTimeout,proto3" json:"stream_timeout,omitempty"`
	// Jitter for upstream connection stream timeouts. Used with timeout to reset streams without overloading
	// the upstream server. Jitter is the upper tolerance for random variation in the timeout. e.g. timeout=15s,
	// jitter=5s -> stream timeout is a random value between 15s and 20s.
	// If unset defaults to no jitter.
	StreamTimeoutJitter string `protobuf:"bytes,4,opt,name=stream_timeout_jitter,json=streamTimeoutJitter,proto3" json:"stream_timeout_jitter,omitempty"`
	// Optional. Maximum timeout for blocked gRPC sends.
	// If this timeout is reached, the stream will be disconnected and re-opened.
	//
	// The value of gRPC send timeout is generally,
	// min(stream_timeout - stream_open_duration, stream_send_max_timeout)
	//
	// A stream_send_min_timeout buffer is added for short final send timeout values to prevent
	// scenarios where the stream deadline is reached too quickly. The complete accuracy of the
	// value is therefore,
	// stream send timeout = max(stream_send_min_timeout, min(stream_timeout - stream_open_duration, stream_send_max_timeout))
	//
	// If unset, the stream_timeout value is used.
	//
	// Examples:
	//
	// Ex 1:
	//   stream_timeout = 15m
	//   stream_send_max_timeout = 5m
	//   stream_send_min_timeout = 1m
	//   ... 1m in send blocks
	//       final send timeout = max(1m, min(15m - 1m, 5m)) = 5m
	//   ... 11m in send blocks
	//       final send timeout = max(1m, min(15m - 11m, 5m)) = 4m
	//   ... 14.5m in send blocks
	//       A 1m buffer is added for short final send timeout values to prevent scenarios where
	//       the stream deadline is reached too quickly.
	//       final send timeout = max(1m, min(15m - 14.5m, 5m)) = 1m
	//
	// Ex 2:
	//   stream_timeout = 5m
	//   stream_send_max_timeout = 10m
	//   stream_send_min_timeout = "" // not configured
	//   ... 1m in send blocks
	//       final send timeout = min(5m - 1m, 10m) = 4m
	//   ... 4m in send blocks
	//       final send timeout = min(5m - 4m, 10m) = 1m
	//   ... > 5m in send blocks will never occur because of the 5m stream timeout
	//
	// Ex 3:
	//   stream_timeout = "" // not configured
	//   stream_send_max_timeout = 5m
	//   stream_send_min_timeout = 4m
	//   ... in all send block scenarios,
	//       final send timeout = max(4m, 5m) = 5m
	//
	// Ex 4:
	//   stream_timeout = 10m
	//   stream_send_max_timeout = "" // not configured
	//   stream_send_min_timeout = 1m
	//   ... in all send block scenarios,
	//       final send timeout = max(1m, 10m) = 10m
	//
	StreamSendMaxTimeout string `protobuf:"bytes,5,opt,name=stream_send_max_timeout,json=streamSendMaxTimeout,proto3" json:"stream_send_max_timeout,omitempty"`
	// Optional. Used in conjunction with stream_send_max_timeout.
	// Refer to document for stream_send_max_timeout.
	// This value must not be > stream_send_max_timeout. Server enforced.
	StreamSendMinTimeout string `protobuf:"bytes,6,opt,name=stream_send_min_timeout,json=streamSendMinTimeout,proto3" json:"stream_send_min_timeout,omitempty"`
	// Optional. Maximum timeout for blocked gRPC recvs.
	// If this timeout is reached, a NACK will be sent to the upstream server with the same
	// version and nounce as the prior attempted send.
	//
	// The value of gRPC recv timeout is generally,
	// min(stream_timeout - stream_open_duration, stream_recv_max_timeout)
	//
	// A stream_recv_min_timeout buffer is added for short final recv timeout values to prevent
	// scenarios where the stream deadline is reached too quickly. The complete accuracy of the
	// value is therefore,
	// stream recv timeout = max(stream_recv_min_timeout, min(stream_timeout - stream_open_duration, stream_recv_max_timeout))
	//
	// If unset, the stream_timeout value is used.
	//
	// Please refer to stream_send_max_timeout for examples, replacing send with recv.
	StreamRecvMaxTimeout string `protobuf:"bytes,7,opt,name=stream_recv_max_timeout,json=streamRecvMaxTimeout,proto3" json:"stream_recv_max_timeout,omitempty"`
	// Optional. Used in conjunction with stream_recv_max_timeout.
	// Refer to document for stream_recv_max_timeout.
	// This value must not be > stream_recv_max_timeout. Server enforced.
	StreamRecvMinTimeout string `protobuf:"bytes,8,opt,name=stream_recv_min_timeout,json=streamRecvMinTimeout,proto3" json:"stream_recv_min_timeout,omitempty"`
}

func (x *Upstream) Reset() {
	*x = Upstream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Upstream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Upstream) ProtoMessage() {}

func (x *Upstream) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Upstream.ProtoReflect.Descriptor instead.
func (*Upstream) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{2}
}

func (x *Upstream) GetAddress() *SocketAddress {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *Upstream) GetKeepAliveTime() string {
	if x != nil {
		return x.KeepAliveTime
	}
	return ""
}

func (x *Upstream) GetStreamTimeout() string {
	if x != nil {
		return x.StreamTimeout
	}
	return ""
}

func (x *Upstream) GetStreamTimeoutJitter() string {
	if x != nil {
		return x.StreamTimeoutJitter
	}
	return ""
}

func (x *Upstream) GetStreamSendMaxTimeout() string {
	if x != nil {
		return x.StreamSendMaxTimeout
	}
	return ""
}

func (x *Upstream) GetStreamSendMinTimeout() string {
	if x != nil {
		return x.StreamSendMinTimeout
	}
	return ""
}

func (x *Upstream) GetStreamRecvMaxTimeout() string {
	if x != nil {
		return x.StreamRecvMaxTimeout
	}
	return ""
}

func (x *Upstream) GetStreamRecvMinTimeout() string {
	if x != nil {
		return x.StreamRecvMinTimeout
	}
	return ""
}

// [#next-free-field: 3]
type Logging struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Filepath where logs are emitted. If no filepath is specified, logs will be written to stderr.
	Path  string        `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Level Logging_Level `protobuf:"varint,2,opt,name=level,proto3,enum=bootstrap.Logging_Level" json:"level,omitempty"`
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

func (x *Logging) GetLevel() Logging_Level {
	if x != nil {
		return x.Level
	}
	return Logging_INFO
}

// [#next-free-field: 3]
type Cache struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Duration before which a key is evicted from the request/response cache. Zero means no expiration time.
	Ttl *duration.Duration `protobuf:"bytes,1,opt,name=ttl,proto3" json:"ttl,omitempty"`
	// The maximum number of keys allowed in the request/response cache. If unset, no maximum number will be enforced.
	MaxEntries int32 `protobuf:"varint,2,opt,name=max_entries,json=maxEntries,proto3" json:"max_entries,omitempty"`
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

func (x *Cache) GetMaxEntries() int32 {
	if x != nil {
		return x.MaxEntries
	}
	return 0
}

// [#next-free-field: 3]
type SocketAddress struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The address for this socket. Listeners will bind to the address.
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

// [#next-free-field: 2]
type Admin struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The TCP address that the admin server will listen on.
	Address *SocketAddress `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *Admin) Reset() {
	*x = Admin{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Admin) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Admin) ProtoMessage() {}

func (x *Admin) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Admin.ProtoReflect.Descriptor instead.
func (*Admin) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{6}
}

func (x *Admin) GetAddress() *SocketAddress {
	if x != nil {
		return x.Address
	}
	return nil
}

// The type of metrics sink, i.e. statsd, prometheus, etc.
type MetricsSink struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//	*MetricsSink_Statsd
	Type isMetricsSink_Type `protobuf_oneof:"type"`
}

func (x *MetricsSink) Reset() {
	*x = MetricsSink{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetricsSink) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetricsSink) ProtoMessage() {}

func (x *MetricsSink) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetricsSink.ProtoReflect.Descriptor instead.
func (*MetricsSink) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{7}
}

func (m *MetricsSink) GetType() isMetricsSink_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *MetricsSink) GetStatsd() *Statsd {
	if x, ok := x.GetType().(*MetricsSink_Statsd); ok {
		return x.Statsd
	}
	return nil
}

type isMetricsSink_Type interface {
	isMetricsSink_Type()
}

type MetricsSink_Statsd struct {
	Statsd *Statsd `protobuf:"bytes,1,opt,name=statsd,proto3,oneof"`
}

func (*MetricsSink_Statsd) isMetricsSink_Type() {}

// [#next-free-field: 4]
type Statsd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address       *SocketAddress     `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	RootPrefix    string             `protobuf:"bytes,2,opt,name=root_prefix,json=rootPrefix,proto3" json:"root_prefix,omitempty"`
	FlushInterval *duration.Duration `protobuf:"bytes,3,opt,name=flush_interval,json=flushInterval,proto3" json:"flush_interval,omitempty"`
}

func (x *Statsd) Reset() {
	*x = Statsd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Statsd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Statsd) ProtoMessage() {}

func (x *Statsd) ProtoReflect() protoreflect.Message {
	mi := &file_bootstrap_v1_bootstrap_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Statsd.ProtoReflect.Descriptor instead.
func (*Statsd) Descriptor() ([]byte, []int) {
	return file_bootstrap_v1_bootstrap_proto_rawDescGZIP(), []int{8}
}

func (x *Statsd) GetAddress() *SocketAddress {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *Statsd) GetRootPrefix() string {
	if x != nil {
		return x.RootPrefix
	}
	return ""
}

func (x *Statsd) GetFlushInterval() *duration.Duration {
	if x != nil {
		return x.FlushInterval
	}
	return nil
}

var File_bootstrap_v1_bootstrap_proto protoreflect.FileDescriptor

var file_bootstrap_v1_bootstrap_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2f, 0x76, 0x31, 0x2f, 0x62,
	0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09,
	0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0xe5, 0x02, 0x0a, 0x09, 0x42, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70,
	0x12, 0x33, 0x0a, 0x06, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x06, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x42, 0x0a, 0x0d, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x5f,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62,
	0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x55, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x0c, 0x6f, 0x72, 0x69,
	0x67, 0x69, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x36, 0x0a, 0x07, 0x6c, 0x6f, 0x67,
	0x67, 0x69, 0x6e, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x6f, 0x6f,
	0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x42, 0x08,
	0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x6c, 0x6f, 0x67, 0x67, 0x69, 0x6e,
	0x67, 0x12, 0x30, 0x0a, 0x05, 0x63, 0x61, 0x63, 0x68, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x10, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x43, 0x61, 0x63,
	0x68, 0x65, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x05, 0x63, 0x61,
	0x63, 0x68, 0x65, 0x12, 0x43, 0x0a, 0x0c, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x73,
	0x69, 0x6e, 0x6b, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x62, 0x6f, 0x6f, 0x74,
	0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x53, 0x69, 0x6e,
	0x6b, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x0b, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x53, 0x69, 0x6e, 0x6b, 0x12, 0x30, 0x0a, 0x05, 0x61, 0x64, 0x6d, 0x69,
	0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74,
	0x72, 0x61, 0x70, 0x2e, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01,
	0x02, 0x10, 0x01, 0x52, 0x05, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x22, 0x46, 0x0a, 0x06, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x12, 0x3c, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61,
	0x70, 0x2e, 0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x42,
	0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x22, 0xa7, 0x03, 0x0a, 0x08, 0x55, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12,
	0x3c, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x53, 0x6f, 0x63,
	0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a,
	0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x26, 0x0a,
	0x0f, 0x6b, 0x65, 0x65, 0x70, 0x5f, 0x61, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6b, 0x65, 0x65, 0x70, 0x41, 0x6c, 0x69, 0x76,
	0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x25, 0x0a, 0x0e, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x73,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x12, 0x32, 0x0a, 0x15,
	0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6a,
	0x69, 0x74, 0x74, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x73, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x4a, 0x69, 0x74, 0x74, 0x65, 0x72,
	0x12, 0x35, 0x0a, 0x17, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x73, 0x65, 0x6e, 0x64, 0x5f,
	0x6d, 0x61, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x14, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x53, 0x65, 0x6e, 0x64, 0x4d, 0x61, 0x78,
	0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x12, 0x35, 0x0a, 0x17, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x5f, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x6d, 0x69, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x53, 0x65, 0x6e, 0x64, 0x4d, 0x69, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x12, 0x35,
	0x0a, 0x17, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f, 0x72, 0x65, 0x63, 0x76, 0x5f, 0x6d, 0x61,
	0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x14, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65, 0x63, 0x76, 0x4d, 0x61, 0x78, 0x54, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x12, 0x35, 0x0a, 0x17, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x5f,
	0x72, 0x65, 0x63, 0x76, 0x5f, 0x6d, 0x69, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74,
	0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65,
	0x63, 0x76, 0x4d, 0x69, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x22, 0x8a, 0x01, 0x0a,
	0x07, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x38, 0x0a, 0x05,
	0x6c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x18, 0x2e, 0x62, 0x6f,
	0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x2e,
	0x4c, 0x65, 0x76, 0x65, 0x6c, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x82, 0x01, 0x02, 0x10, 0x01, 0x52,
	0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x22, 0x31, 0x0a, 0x05, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x12,
	0x08, 0x0a, 0x04, 0x49, 0x4e, 0x46, 0x4f, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x44, 0x45, 0x42,
	0x55, 0x47, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x57, 0x41, 0x52, 0x4e, 0x10, 0x02, 0x12, 0x09,
	0x0a, 0x05, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x03, 0x22, 0x61, 0x0a, 0x05, 0x43, 0x61, 0x63,
	0x68, 0x65, 0x12, 0x37, 0x0a, 0x03, 0x74, 0x74, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0xaa,
	0x01, 0x04, 0x08, 0x01, 0x32, 0x00, 0x52, 0x03, 0x74, 0x74, 0x6c, 0x12, 0x1f, 0x0a, 0x0b, 0x6d,
	0x61, 0x78, 0x5f, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x0a, 0x6d, 0x61, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x22, 0x5d, 0x0a, 0x0d,
	0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x22, 0x0a,
	0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08,
	0xfa, 0x42, 0x05, 0x72, 0x03, 0xa8, 0x01, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x12, 0x28, 0x0a, 0x0a, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x09, 0xfa, 0x42, 0x06, 0x2a, 0x04, 0x18, 0xff, 0xff, 0x03,
	0x52, 0x09, 0x70, 0x6f, 0x72, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x45, 0x0a, 0x05, 0x41,
	0x64, 0x6d, 0x69, 0x6e, 0x12, 0x3c, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61,
	0x70, 0x2e, 0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x42,
	0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x22, 0x47, 0x0a, 0x0b, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x53, 0x69, 0x6e,
	0x6b, 0x12, 0x2b, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x73, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x73, 0x64, 0x48, 0x00, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x73, 0x64, 0x42, 0x0b,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x03, 0xf8, 0x42, 0x01, 0x22, 0xbe, 0x01, 0x0a, 0x06,
	0x53, 0x74, 0x61, 0x74, 0x73, 0x64, 0x12, 0x3c, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74,
	0x72, 0x61, 0x70, 0x2e, 0x53, 0x6f, 0x63, 0x6b, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x07, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x12, 0x28, 0x0a, 0x0b, 0x72, 0x6f, 0x6f, 0x74, 0x5f, 0x70, 0x72, 0x65,
	0x66, 0x69, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x72, 0x02,
	0x20, 0x01, 0x52, 0x0a, 0x72, 0x6f, 0x6f, 0x74, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x12, 0x4c,
	0x0a, 0x0e, 0x66, 0x6c, 0x75, 0x73, 0x68, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0xaa, 0x01, 0x04, 0x08, 0x01, 0x32, 0x00, 0x52, 0x0d, 0x66,
	0x6c, 0x75, 0x73, 0x68, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x42, 0x1a, 0x5a, 0x18,
	0x62, 0x6f, 0x6f, 0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x2f, 0x76, 0x31, 0x3b, 0x62, 0x6f, 0x6f,
	0x74, 0x73, 0x74, 0x72, 0x61, 0x70, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_bootstrap_v1_bootstrap_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_bootstrap_v1_bootstrap_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_bootstrap_v1_bootstrap_proto_goTypes = []interface{}{
	(Logging_Level)(0),        // 0: bootstrap.Logging.Level
	(*Bootstrap)(nil),         // 1: bootstrap.Bootstrap
	(*Server)(nil),            // 2: bootstrap.Server
	(*Upstream)(nil),          // 3: bootstrap.Upstream
	(*Logging)(nil),           // 4: bootstrap.Logging
	(*Cache)(nil),             // 5: bootstrap.Cache
	(*SocketAddress)(nil),     // 6: bootstrap.SocketAddress
	(*Admin)(nil),             // 7: bootstrap.Admin
	(*MetricsSink)(nil),       // 8: bootstrap.MetricsSink
	(*Statsd)(nil),            // 9: bootstrap.Statsd
	(*duration.Duration)(nil), // 10: google.protobuf.Duration
}
var file_bootstrap_v1_bootstrap_proto_depIdxs = []int32{
	2,  // 0: bootstrap.Bootstrap.server:type_name -> bootstrap.Server
	3,  // 1: bootstrap.Bootstrap.origin_server:type_name -> bootstrap.Upstream
	4,  // 2: bootstrap.Bootstrap.logging:type_name -> bootstrap.Logging
	5,  // 3: bootstrap.Bootstrap.cache:type_name -> bootstrap.Cache
	8,  // 4: bootstrap.Bootstrap.metrics_sink:type_name -> bootstrap.MetricsSink
	7,  // 5: bootstrap.Bootstrap.admin:type_name -> bootstrap.Admin
	6,  // 6: bootstrap.Server.address:type_name -> bootstrap.SocketAddress
	6,  // 7: bootstrap.Upstream.address:type_name -> bootstrap.SocketAddress
	0,  // 8: bootstrap.Logging.level:type_name -> bootstrap.Logging.Level
	10, // 9: bootstrap.Cache.ttl:type_name -> google.protobuf.Duration
	6,  // 10: bootstrap.Admin.address:type_name -> bootstrap.SocketAddress
	9,  // 11: bootstrap.MetricsSink.statsd:type_name -> bootstrap.Statsd
	6,  // 12: bootstrap.Statsd.address:type_name -> bootstrap.SocketAddress
	10, // 13: bootstrap.Statsd.flush_interval:type_name -> google.protobuf.Duration
	14, // [14:14] is the sub-list for method output_type
	14, // [14:14] is the sub-list for method input_type
	14, // [14:14] is the sub-list for extension type_name
	14, // [14:14] is the sub-list for extension extendee
	0,  // [0:14] is the sub-list for field type_name
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
			switch v := v.(*Upstream); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Admin); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetricsSink); i {
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
		file_bootstrap_v1_bootstrap_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Statsd); i {
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
	file_bootstrap_v1_bootstrap_proto_msgTypes[7].OneofWrappers = []interface{}{
		(*MetricsSink_Statsd)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_bootstrap_v1_bootstrap_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_bootstrap_v1_bootstrap_proto_goTypes,
		DependencyIndexes: file_bootstrap_v1_bootstrap_proto_depIdxs,
		EnumInfos:         file_bootstrap_v1_bootstrap_proto_enumTypes,
		MessageInfos:      file_bootstrap_v1_bootstrap_proto_msgTypes,
	}.Build()
	File_bootstrap_v1_bootstrap_proto = out.File
	file_bootstrap_v1_bootstrap_proto_rawDesc = nil
	file_bootstrap_v1_bootstrap_proto_goTypes = nil
	file_bootstrap_v1_bootstrap_proto_depIdxs = nil
}
