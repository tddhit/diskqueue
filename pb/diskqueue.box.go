// Code generated by protoc-gen-box. DO NOT EDIT.
// source: diskqueue.proto

package diskqueue

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	tr "github.com/tddhit/box/transport"
	tropt "github.com/tddhit/box/transport/option"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

import (
	context1 "golang.org/x/net/context"
	grpc1 "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Message struct {
	ID                   uint64   `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Data                 []byte   `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Timestamp            int64    `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{0}
}
func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (dst *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(dst, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetID() uint64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *Message) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Message) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

type PublishRequest struct {
	Topic                string   `protobuf:"bytes,1,opt,name=topic,proto3" json:"topic,omitempty"`
	Data                 []byte   `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	HashKey              []byte   `protobuf:"bytes,3,opt,name=hashKey,proto3" json:"hashKey,omitempty"`
	IgnoreFilter         bool     `protobuf:"varint,4,opt,name=ignoreFilter,proto3" json:"ignoreFilter,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PublishRequest) Reset()         { *m = PublishRequest{} }
func (m *PublishRequest) String() string { return proto.CompactTextString(m) }
func (*PublishRequest) ProtoMessage()    {}
func (*PublishRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{1}
}
func (m *PublishRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PublishRequest.Unmarshal(m, b)
}
func (m *PublishRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PublishRequest.Marshal(b, m, deterministic)
}
func (dst *PublishRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PublishRequest.Merge(dst, src)
}
func (m *PublishRequest) XXX_Size() int {
	return xxx_messageInfo_PublishRequest.Size(m)
}
func (m *PublishRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PublishRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PublishRequest proto.InternalMessageInfo

func (m *PublishRequest) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

func (m *PublishRequest) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *PublishRequest) GetHashKey() []byte {
	if m != nil {
		return m.HashKey
	}
	return nil
}

func (m *PublishRequest) GetIgnoreFilter() bool {
	if m != nil {
		return m.IgnoreFilter
	}
	return false
}

type PublishReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PublishReply) Reset()         { *m = PublishReply{} }
func (m *PublishReply) String() string { return proto.CompactTextString(m) }
func (*PublishReply) ProtoMessage()    {}
func (*PublishReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{2}
}
func (m *PublishReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PublishReply.Unmarshal(m, b)
}
func (m *PublishReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PublishReply.Marshal(b, m, deterministic)
}
func (dst *PublishReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PublishReply.Merge(dst, src)
}
func (m *PublishReply) XXX_Size() int {
	return xxx_messageInfo_PublishReply.Size(m)
}
func (m *PublishReply) XXX_DiscardUnknown() {
	xxx_messageInfo_PublishReply.DiscardUnknown(m)
}

var xxx_messageInfo_PublishReply proto.InternalMessageInfo

type PullRequest struct {
	Topic                string   `protobuf:"bytes,1,opt,name=topic,proto3" json:"topic,omitempty"`
	NeedAck              bool     `protobuf:"varint,2,opt,name=needAck,proto3" json:"needAck,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PullRequest) Reset()         { *m = PullRequest{} }
func (m *PullRequest) String() string { return proto.CompactTextString(m) }
func (*PullRequest) ProtoMessage()    {}
func (*PullRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{3}
}
func (m *PullRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PullRequest.Unmarshal(m, b)
}
func (m *PullRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PullRequest.Marshal(b, m, deterministic)
}
func (dst *PullRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PullRequest.Merge(dst, src)
}
func (m *PullRequest) XXX_Size() int {
	return xxx_messageInfo_PullRequest.Size(m)
}
func (m *PullRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PullRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PullRequest proto.InternalMessageInfo

func (m *PullRequest) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

func (m *PullRequest) GetNeedAck() bool {
	if m != nil {
		return m.NeedAck
	}
	return false
}

type PullReply struct {
	Message              *Message `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PullReply) Reset()         { *m = PullReply{} }
func (m *PullReply) String() string { return proto.CompactTextString(m) }
func (*PullReply) ProtoMessage()    {}
func (*PullReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{4}
}
func (m *PullReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PullReply.Unmarshal(m, b)
}
func (m *PullReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PullReply.Marshal(b, m, deterministic)
}
func (dst *PullReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PullReply.Merge(dst, src)
}
func (m *PullReply) XXX_Size() int {
	return xxx_messageInfo_PullReply.Size(m)
}
func (m *PullReply) XXX_DiscardUnknown() {
	xxx_messageInfo_PullReply.DiscardUnknown(m)
}

var xxx_messageInfo_PullReply proto.InternalMessageInfo

func (m *PullReply) GetMessage() *Message {
	if m != nil {
		return m.Message
	}
	return nil
}

type AckRequest struct {
	Topic                string   `protobuf:"bytes,1,opt,name=topic,proto3" json:"topic,omitempty"`
	MsgID                uint64   `protobuf:"varint,2,opt,name=msgID,proto3" json:"msgID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AckRequest) Reset()         { *m = AckRequest{} }
func (m *AckRequest) String() string { return proto.CompactTextString(m) }
func (*AckRequest) ProtoMessage()    {}
func (*AckRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{5}
}
func (m *AckRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AckRequest.Unmarshal(m, b)
}
func (m *AckRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AckRequest.Marshal(b, m, deterministic)
}
func (dst *AckRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AckRequest.Merge(dst, src)
}
func (m *AckRequest) XXX_Size() int {
	return xxx_messageInfo_AckRequest.Size(m)
}
func (m *AckRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AckRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AckRequest proto.InternalMessageInfo

func (m *AckRequest) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

func (m *AckRequest) GetMsgID() uint64 {
	if m != nil {
		return m.MsgID
	}
	return 0
}

type AckReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AckReply) Reset()         { *m = AckReply{} }
func (m *AckReply) String() string { return proto.CompactTextString(m) }
func (*AckReply) ProtoMessage()    {}
func (*AckReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_17490c04a98e3f89, []int{6}
}
func (m *AckReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AckReply.Unmarshal(m, b)
}
func (m *AckReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AckReply.Marshal(b, m, deterministic)
}
func (dst *AckReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AckReply.Merge(dst, src)
}
func (m *AckReply) XXX_Size() int {
	return xxx_messageInfo_AckReply.Size(m)
}
func (m *AckReply) XXX_DiscardUnknown() {
	xxx_messageInfo_AckReply.DiscardUnknown(m)
}

var xxx_messageInfo_AckReply proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Message)(nil), "diskqueue.Message")
	proto.RegisterType((*PublishRequest)(nil), "diskqueue.PublishRequest")
	proto.RegisterType((*PublishReply)(nil), "diskqueue.PublishReply")
	proto.RegisterType((*PullRequest)(nil), "diskqueue.PullRequest")
	proto.RegisterType((*PullReply)(nil), "diskqueue.PullReply")
	proto.RegisterType((*AckRequest)(nil), "diskqueue.AckRequest")
	proto.RegisterType((*AckReply)(nil), "diskqueue.AckReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ tr.Server
var _ tr.ClientConn
var _ tropt.CallOption

type DiskqueueGrpcClient interface {
	Publish(ctx context.Context, in *PublishRequest, opts ...tropt.CallOption) (*PublishReply, error)
	MPublish(ctx context.Context, opts ...tropt.CallOption) (Diskqueue_MPublishClient, error)
	Pull(ctx context.Context, in *PullRequest, opts ...tropt.CallOption) (*PullReply, error)
	Ack(ctx context.Context, in *AckRequest, opts ...tropt.CallOption) (*AckReply, error)
}

type diskqueueGrpcClient struct {
	cc tr.ClientConn
}

func NewDiskqueueGrpcClient(cc tr.ClientConn) DiskqueueGrpcClient {
	return &diskqueueGrpcClient{cc}
}

func (c *diskqueueGrpcClient) Publish(ctx context.Context, in *PublishRequest, opts ...tropt.CallOption) (*PublishReply, error) {
	out := new(PublishReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Publish", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueGrpcClient) MPublish(ctx context.Context, opts ...tropt.CallOption) (Diskqueue_MPublishClient, error) {
	stream, err := c.cc.NewStream(ctx, DiskqueueGrpcServiceDesc, 0, "/diskqueue.Diskqueue/MPublish", opts...)
	if err != nil {
		return nil, err
	}
	x := &diskqueueMPublishClient{stream}
	return x, nil
}

func (c *diskqueueGrpcClient) Pull(ctx context.Context, in *PullRequest, opts ...tropt.CallOption) (*PullReply, error) {
	out := new(PullReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Pull", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueGrpcClient) Ack(ctx context.Context, in *AckRequest, opts ...tropt.CallOption) (*AckReply, error) {
	out := new(AckReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Ack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type diskqueueGrpcServiceDesc struct {
	desc *grpc.ServiceDesc
}

func (d *diskqueueGrpcServiceDesc) Desc() interface{} {
	return d.desc
}

var DiskqueueGrpcServiceDesc = &diskqueueGrpcServiceDesc{&_Diskqueue_serviceDesc}

// Reference imports to suppress errors if they are not otherwise used.
var _ context1.Context
var _ grpc1.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc1.SupportPackageIsVersion4

// DiskqueueClient is the client API for Diskqueue service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DiskqueueClient interface {
	Publish(ctx context1.Context, in *PublishRequest, opts ...grpc1.CallOption) (*PublishReply, error)
	MPublish(ctx context1.Context, opts ...grpc1.CallOption) (Diskqueue_MPublishClient, error)
	Pull(ctx context1.Context, in *PullRequest, opts ...grpc1.CallOption) (*PullReply, error)
	Ack(ctx context1.Context, in *AckRequest, opts ...grpc1.CallOption) (*AckReply, error)
}

type diskqueueClient struct {
	cc *grpc1.ClientConn
}

func NewDiskqueueClient(cc *grpc1.ClientConn) DiskqueueClient {
	return &diskqueueClient{cc}
}

func (c *diskqueueClient) Publish(ctx context1.Context, in *PublishRequest, opts ...grpc1.CallOption) (*PublishReply, error) {
	out := new(PublishReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Publish", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) MPublish(ctx context1.Context, opts ...grpc1.CallOption) (Diskqueue_MPublishClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Diskqueue_serviceDesc.Streams[0], "/diskqueue.Diskqueue/MPublish", opts...)
	if err != nil {
		return nil, err
	}
	x := &diskqueueMPublishClient{stream}
	return x, nil
}

type Diskqueue_MPublishClient interface {
	Send(*PublishRequest) error
	CloseAndRecv() (*PublishReply, error)
	grpc1.ClientStream
}

type diskqueueMPublishClient struct {
	grpc1.ClientStream
}

func (x *diskqueueMPublishClient) Send(m *PublishRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *diskqueueMPublishClient) CloseAndRecv() (*PublishReply, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(PublishReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *diskqueueClient) Pull(ctx context1.Context, in *PullRequest, opts ...grpc1.CallOption) (*PullReply, error) {
	out := new(PullReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Pull", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) Ack(ctx context1.Context, in *AckRequest, opts ...grpc1.CallOption) (*AckReply, error) {
	out := new(AckReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Ack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DiskqueueServer is the server API for Diskqueue service.
type DiskqueueServer interface {
	Publish(context1.Context, *PublishRequest) (*PublishReply, error)
	MPublish(Diskqueue_MPublishServer) error
	Pull(context1.Context, *PullRequest) (*PullReply, error)
	Ack(context1.Context, *AckRequest) (*AckReply, error)
}

func RegisterDiskqueueServer(s *grpc1.Server, srv DiskqueueServer) {
	s.RegisterService(&_Diskqueue_serviceDesc, srv)
}

func _Diskqueue_Publish_Handler(srv interface{}, ctx context1.Context, dec func(interface{}) error, interceptor grpc1.UnaryServerInterceptor) (interface{}, error) {
	in := new(PublishRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Publish(ctx, in)
	}
	info := &grpc1.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Publish",
	}
	handler := func(ctx context1.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Publish(ctx, req.(*PublishRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_MPublish_Handler(srv interface{}, stream grpc1.ServerStream) error {
	return srv.(DiskqueueServer).MPublish(&diskqueueMPublishServer{stream})
}

type Diskqueue_MPublishServer interface {
	SendAndClose(*PublishReply) error
	Recv() (*PublishRequest, error)
	grpc1.ServerStream
}

type diskqueueMPublishServer struct {
	grpc1.ServerStream
}

func (x *diskqueueMPublishServer) SendAndClose(m *PublishReply) error {
	return x.ServerStream.SendMsg(m)
}

func (x *diskqueueMPublishServer) Recv() (*PublishRequest, error) {
	m := new(PublishRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Diskqueue_Pull_Handler(srv interface{}, ctx context1.Context, dec func(interface{}) error, interceptor grpc1.UnaryServerInterceptor) (interface{}, error) {
	in := new(PullRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Pull(ctx, in)
	}
	info := &grpc1.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Pull",
	}
	handler := func(ctx context1.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Pull(ctx, req.(*PullRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_Ack_Handler(srv interface{}, ctx context1.Context, dec func(interface{}) error, interceptor grpc1.UnaryServerInterceptor) (interface{}, error) {
	in := new(AckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Ack(ctx, in)
	}
	info := &grpc1.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Ack",
	}
	handler := func(ctx context1.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Ack(ctx, req.(*AckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Diskqueue_serviceDesc = grpc1.ServiceDesc{
	ServiceName: "diskqueue.Diskqueue",
	HandlerType: (*DiskqueueServer)(nil),
	Methods: []grpc1.MethodDesc{
		{
			MethodName: "Publish",
			Handler:    _Diskqueue_Publish_Handler,
		},
		{
			MethodName: "Pull",
			Handler:    _Diskqueue_Pull_Handler,
		},
		{
			MethodName: "Ack",
			Handler:    _Diskqueue_Ack_Handler,
		},
	},
	Streams: []grpc1.StreamDesc{
		{
			StreamName:    "MPublish",
			Handler:       _Diskqueue_MPublish_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "diskqueue.proto",
}

func init() { proto.RegisterFile("diskqueue.proto", fileDescriptor_diskqueue_17490c04a98e3f89) }

var fileDescriptor_diskqueue_17490c04a98e3f89 = []byte{
	// 348 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x92, 0x4f, 0x6b, 0xea, 0x40,
	0x14, 0xc5, 0xdf, 0x68, 0x7c, 0x49, 0xae, 0xc1, 0x07, 0xf7, 0xf9, 0x5e, 0x53, 0xe9, 0x22, 0xcc,
	0x2a, 0x8b, 0xe2, 0x42, 0xa1, 0xb4, 0x8b, 0x52, 0x2c, 0xa1, 0x20, 0x22, 0xc8, 0x7c, 0x83, 0x18,
	0x07, 0x0d, 0x99, 0x98, 0xe8, 0x4c, 0xa0, 0xf9, 0xf0, 0x85, 0xe2, 0xc4, 0xf8, 0xa7, 0x15, 0x37,
	0xdd, 0xe5, 0x9e, 0x99, 0x73, 0xce, 0xcd, 0x8f, 0x81, 0x3f, 0x8b, 0x58, 0x26, 0x9b, 0x82, 0x17,
	0xbc, 0x9f, 0x6f, 0x33, 0x95, 0xa1, 0x7d, 0x10, 0xe8, 0x04, 0xcc, 0x29, 0x97, 0x32, 0x5c, 0x72,
	0xec, 0x40, 0x63, 0x1c, 0xb8, 0xc4, 0x23, 0xbe, 0xc1, 0x1a, 0xe3, 0x00, 0x11, 0x8c, 0x45, 0xa8,
	0x42, 0xb7, 0xe1, 0x11, 0xdf, 0x61, 0xfa, 0x1b, 0xef, 0xc0, 0x56, 0x71, 0xca, 0xa5, 0x0a, 0xd3,
	0xdc, 0x6d, 0x7a, 0xc4, 0x6f, 0xb2, 0xa3, 0x40, 0xdf, 0xa1, 0x33, 0x2b, 0xe6, 0x22, 0x96, 0x2b,
	0xc6, 0x37, 0x05, 0x97, 0x0a, 0xbb, 0xd0, 0x52, 0x59, 0x1e, 0x47, 0x3a, 0xd6, 0x66, 0xd5, 0x70,
	0x31, 0xd9, 0x05, 0x73, 0x15, 0xca, 0xd5, 0x84, 0x97, 0x3a, 0xd7, 0x61, 0xf5, 0x88, 0x14, 0x9c,
	0x78, 0xb9, 0xce, 0xb6, 0xfc, 0x2d, 0x16, 0x8a, 0x6f, 0x5d, 0xc3, 0x23, 0xbe, 0xc5, 0xce, 0x34,
	0xda, 0x01, 0xe7, 0xd0, 0x9c, 0x8b, 0x92, 0x3e, 0x43, 0x7b, 0x56, 0x08, 0x71, 0x7d, 0x0d, 0x17,
	0xcc, 0x35, 0xe7, 0x8b, 0x51, 0x94, 0xe8, 0x4d, 0x2c, 0x56, 0x8f, 0xf4, 0x09, 0xec, 0xca, 0x9e,
	0x8b, 0x12, 0xef, 0xc1, 0x4c, 0x2b, 0x44, 0xda, 0xde, 0x1e, 0x60, 0xff, 0x08, 0x74, 0x0f, 0x8f,
	0xd5, 0x57, 0xe8, 0x23, 0xc0, 0x28, 0x4a, 0xae, 0x17, 0x77, 0xa1, 0x95, 0xca, 0xe5, 0x38, 0xd0,
	0xb5, 0x06, 0xab, 0x06, 0x0a, 0x60, 0x69, 0x67, 0x2e, 0xca, 0xc1, 0x07, 0x01, 0x3b, 0xa8, 0x4b,
	0xf0, 0x05, 0xcc, 0xfd, 0xdf, 0xe1, 0xed, 0x49, 0xf7, 0x39, 0xeb, 0xde, 0xcd, 0xa5, 0xa3, 0x1d,
	0x8c, 0x5f, 0xf8, 0x0a, 0xd6, 0xf4, 0x47, 0x09, 0x3e, 0xc1, 0x07, 0x30, 0x76, 0x4c, 0xf0, 0xff,
	0xd9, 0xa5, 0x03, 0xe3, 0x5e, 0xf7, 0x9b, 0x5e, 0x75, 0x0f, 0xa1, 0x39, 0x8a, 0x12, 0xfc, 0x77,
	0x72, 0x7c, 0x04, 0xd4, 0xfb, 0xfb, 0x55, 0xd6, 0xa6, 0xf9, 0x6f, 0xfd, 0x50, 0x87, 0x9f, 0x01,
	0x00, 0x00, 0xff, 0xff, 0x4a, 0x56, 0x6e, 0xcd, 0xbb, 0x02, 0x00, 0x00,
}
