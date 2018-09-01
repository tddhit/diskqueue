// Code generated by protoc-gen-go. DO NOT EDIT.
// source: diskqueue.proto

package diskqueue

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
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
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{0}
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
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PublishRequest) Reset()         { *m = PublishRequest{} }
func (m *PublishRequest) String() string { return proto.CompactTextString(m) }
func (*PublishRequest) ProtoMessage()    {}
func (*PublishRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{1}
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

type PublishReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PublishReply) Reset()         { *m = PublishReply{} }
func (m *PublishReply) String() string { return proto.CompactTextString(m) }
func (*PublishReply) ProtoMessage()    {}
func (*PublishReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{2}
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

type SubscribeRequest struct {
	Topic                string   `protobuf:"bytes,1,opt,name=topic,proto3" json:"topic,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SubscribeRequest) Reset()         { *m = SubscribeRequest{} }
func (m *SubscribeRequest) String() string { return proto.CompactTextString(m) }
func (*SubscribeRequest) ProtoMessage()    {}
func (*SubscribeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{3}
}
func (m *SubscribeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SubscribeRequest.Unmarshal(m, b)
}
func (m *SubscribeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SubscribeRequest.Marshal(b, m, deterministic)
}
func (dst *SubscribeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SubscribeRequest.Merge(dst, src)
}
func (m *SubscribeRequest) XXX_Size() int {
	return xxx_messageInfo_SubscribeRequest.Size(m)
}
func (m *SubscribeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SubscribeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SubscribeRequest proto.InternalMessageInfo

func (m *SubscribeRequest) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

type SubscribeReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SubscribeReply) Reset()         { *m = SubscribeReply{} }
func (m *SubscribeReply) String() string { return proto.CompactTextString(m) }
func (*SubscribeReply) ProtoMessage()    {}
func (*SubscribeReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{4}
}
func (m *SubscribeReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SubscribeReply.Unmarshal(m, b)
}
func (m *SubscribeReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SubscribeReply.Marshal(b, m, deterministic)
}
func (dst *SubscribeReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SubscribeReply.Merge(dst, src)
}
func (m *SubscribeReply) XXX_Size() int {
	return xxx_messageInfo_SubscribeReply.Size(m)
}
func (m *SubscribeReply) XXX_DiscardUnknown() {
	xxx_messageInfo_SubscribeReply.DiscardUnknown(m)
}

var xxx_messageInfo_SubscribeReply proto.InternalMessageInfo

type CancelRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CancelRequest) Reset()         { *m = CancelRequest{} }
func (m *CancelRequest) String() string { return proto.CompactTextString(m) }
func (*CancelRequest) ProtoMessage()    {}
func (*CancelRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{5}
}
func (m *CancelRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CancelRequest.Unmarshal(m, b)
}
func (m *CancelRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CancelRequest.Marshal(b, m, deterministic)
}
func (dst *CancelRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CancelRequest.Merge(dst, src)
}
func (m *CancelRequest) XXX_Size() int {
	return xxx_messageInfo_CancelRequest.Size(m)
}
func (m *CancelRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CancelRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CancelRequest proto.InternalMessageInfo

type CancelReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CancelReply) Reset()         { *m = CancelReply{} }
func (m *CancelReply) String() string { return proto.CompactTextString(m) }
func (*CancelReply) ProtoMessage()    {}
func (*CancelReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{6}
}
func (m *CancelReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CancelReply.Unmarshal(m, b)
}
func (m *CancelReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CancelReply.Marshal(b, m, deterministic)
}
func (dst *CancelReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CancelReply.Merge(dst, src)
}
func (m *CancelReply) XXX_Size() int {
	return xxx_messageInfo_CancelReply.Size(m)
}
func (m *CancelReply) XXX_DiscardUnknown() {
	xxx_messageInfo_CancelReply.DiscardUnknown(m)
}

var xxx_messageInfo_CancelReply proto.InternalMessageInfo

type PullRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PullRequest) Reset()         { *m = PullRequest{} }
func (m *PullRequest) String() string { return proto.CompactTextString(m) }
func (*PullRequest) ProtoMessage()    {}
func (*PullRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{7}
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
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{8}
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

type KeepAliveRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *KeepAliveRequest) Reset()         { *m = KeepAliveRequest{} }
func (m *KeepAliveRequest) String() string { return proto.CompactTextString(m) }
func (*KeepAliveRequest) ProtoMessage()    {}
func (*KeepAliveRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{9}
}
func (m *KeepAliveRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeepAliveRequest.Unmarshal(m, b)
}
func (m *KeepAliveRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeepAliveRequest.Marshal(b, m, deterministic)
}
func (dst *KeepAliveRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeepAliveRequest.Merge(dst, src)
}
func (m *KeepAliveRequest) XXX_Size() int {
	return xxx_messageInfo_KeepAliveRequest.Size(m)
}
func (m *KeepAliveRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_KeepAliveRequest.DiscardUnknown(m)
}

var xxx_messageInfo_KeepAliveRequest proto.InternalMessageInfo

type KeepAliveReply struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *KeepAliveReply) Reset()         { *m = KeepAliveReply{} }
func (m *KeepAliveReply) String() string { return proto.CompactTextString(m) }
func (*KeepAliveReply) ProtoMessage()    {}
func (*KeepAliveReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{10}
}
func (m *KeepAliveReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeepAliveReply.Unmarshal(m, b)
}
func (m *KeepAliveReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeepAliveReply.Marshal(b, m, deterministic)
}
func (dst *KeepAliveReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeepAliveReply.Merge(dst, src)
}
func (m *KeepAliveReply) XXX_Size() int {
	return xxx_messageInfo_KeepAliveReply.Size(m)
}
func (m *KeepAliveReply) XXX_DiscardUnknown() {
	xxx_messageInfo_KeepAliveReply.DiscardUnknown(m)
}

var xxx_messageInfo_KeepAliveReply proto.InternalMessageInfo

type AckRequest struct {
	MsgID                uint64   `protobuf:"varint,1,opt,name=msgID,proto3" json:"msgID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AckRequest) Reset()         { *m = AckRequest{} }
func (m *AckRequest) String() string { return proto.CompactTextString(m) }
func (*AckRequest) ProtoMessage()    {}
func (*AckRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{11}
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
	return fileDescriptor_diskqueue_3971262521e2d53f, []int{12}
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
	proto.RegisterType((*SubscribeRequest)(nil), "diskqueue.SubscribeRequest")
	proto.RegisterType((*SubscribeReply)(nil), "diskqueue.SubscribeReply")
	proto.RegisterType((*CancelRequest)(nil), "diskqueue.CancelRequest")
	proto.RegisterType((*CancelReply)(nil), "diskqueue.CancelReply")
	proto.RegisterType((*PullRequest)(nil), "diskqueue.PullRequest")
	proto.RegisterType((*PullReply)(nil), "diskqueue.PullReply")
	proto.RegisterType((*KeepAliveRequest)(nil), "diskqueue.KeepAliveRequest")
	proto.RegisterType((*KeepAliveReply)(nil), "diskqueue.KeepAliveReply")
	proto.RegisterType((*AckRequest)(nil), "diskqueue.AckRequest")
	proto.RegisterType((*AckReply)(nil), "diskqueue.AckReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// DiskqueueClient is the client API for Diskqueue service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DiskqueueClient interface {
	Publish(ctx context.Context, in *PublishRequest, opts ...grpc.CallOption) (*PublishReply, error)
	MPublish(ctx context.Context, opts ...grpc.CallOption) (Diskqueue_MPublishClient, error)
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (*SubscribeReply, error)
	Cancel(ctx context.Context, in *CancelRequest, opts ...grpc.CallOption) (*CancelReply, error)
	Pull(ctx context.Context, in *PullRequest, opts ...grpc.CallOption) (*PullReply, error)
	KeepAlive(ctx context.Context, in *KeepAliveRequest, opts ...grpc.CallOption) (Diskqueue_KeepAliveClient, error)
	Ack(ctx context.Context, in *AckRequest, opts ...grpc.CallOption) (*AckReply, error)
}

type diskqueueClient struct {
	cc *grpc.ClientConn
}

func NewDiskqueueClient(cc *grpc.ClientConn) DiskqueueClient {
	return &diskqueueClient{cc}
}

func (c *diskqueueClient) Publish(ctx context.Context, in *PublishRequest, opts ...grpc.CallOption) (*PublishReply, error) {
	out := new(PublishReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Publish", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) MPublish(ctx context.Context, opts ...grpc.CallOption) (Diskqueue_MPublishClient, error) {
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
	grpc.ClientStream
}

type diskqueueMPublishClient struct {
	grpc.ClientStream
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

func (c *diskqueueClient) Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (*SubscribeReply, error) {
	out := new(SubscribeReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Subscribe", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) Cancel(ctx context.Context, in *CancelRequest, opts ...grpc.CallOption) (*CancelReply, error) {
	out := new(CancelReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Cancel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) Pull(ctx context.Context, in *PullRequest, opts ...grpc.CallOption) (*PullReply, error) {
	out := new(PullReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Pull", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diskqueueClient) KeepAlive(ctx context.Context, in *KeepAliveRequest, opts ...grpc.CallOption) (Diskqueue_KeepAliveClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Diskqueue_serviceDesc.Streams[1], "/diskqueue.Diskqueue/KeepAlive", opts...)
	if err != nil {
		return nil, err
	}
	x := &diskqueueKeepAliveClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Diskqueue_KeepAliveClient interface {
	Recv() (*KeepAliveReply, error)
	grpc.ClientStream
}

type diskqueueKeepAliveClient struct {
	grpc.ClientStream
}

func (x *diskqueueKeepAliveClient) Recv() (*KeepAliveReply, error) {
	m := new(KeepAliveReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *diskqueueClient) Ack(ctx context.Context, in *AckRequest, opts ...grpc.CallOption) (*AckReply, error) {
	out := new(AckReply)
	err := c.cc.Invoke(ctx, "/diskqueue.Diskqueue/Ack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DiskqueueServer is the server API for Diskqueue service.
type DiskqueueServer interface {
	Publish(context.Context, *PublishRequest) (*PublishReply, error)
	MPublish(Diskqueue_MPublishServer) error
	Subscribe(context.Context, *SubscribeRequest) (*SubscribeReply, error)
	Cancel(context.Context, *CancelRequest) (*CancelReply, error)
	Pull(context.Context, *PullRequest) (*PullReply, error)
	KeepAlive(*KeepAliveRequest, Diskqueue_KeepAliveServer) error
	Ack(context.Context, *AckRequest) (*AckReply, error)
}

func RegisterDiskqueueServer(s *grpc.Server, srv DiskqueueServer) {
	s.RegisterService(&_Diskqueue_serviceDesc, srv)
}

func _Diskqueue_Publish_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PublishRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Publish(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Publish",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Publish(ctx, req.(*PublishRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_MPublish_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DiskqueueServer).MPublish(&diskqueueMPublishServer{stream})
}

type Diskqueue_MPublishServer interface {
	SendAndClose(*PublishReply) error
	Recv() (*PublishRequest, error)
	grpc.ServerStream
}

type diskqueueMPublishServer struct {
	grpc.ServerStream
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

func _Diskqueue_Subscribe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubscribeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Subscribe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Subscribe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Subscribe(ctx, req.(*SubscribeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_Cancel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Cancel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Cancel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Cancel(ctx, req.(*CancelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_Pull_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PullRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Pull(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Pull",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Pull(ctx, req.(*PullRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diskqueue_KeepAlive_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(KeepAliveRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DiskqueueServer).KeepAlive(m, &diskqueueKeepAliveServer{stream})
}

type Diskqueue_KeepAliveServer interface {
	Send(*KeepAliveReply) error
	grpc.ServerStream
}

type diskqueueKeepAliveServer struct {
	grpc.ServerStream
}

func (x *diskqueueKeepAliveServer) Send(m *KeepAliveReply) error {
	return x.ServerStream.SendMsg(m)
}

func _Diskqueue_Ack_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiskqueueServer).Ack(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/diskqueue.Diskqueue/Ack",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiskqueueServer).Ack(ctx, req.(*AckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Diskqueue_serviceDesc = grpc.ServiceDesc{
	ServiceName: "diskqueue.Diskqueue",
	HandlerType: (*DiskqueueServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Publish",
			Handler:    _Diskqueue_Publish_Handler,
		},
		{
			MethodName: "Subscribe",
			Handler:    _Diskqueue_Subscribe_Handler,
		},
		{
			MethodName: "Cancel",
			Handler:    _Diskqueue_Cancel_Handler,
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
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "MPublish",
			Handler:       _Diskqueue_MPublish_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "KeepAlive",
			Handler:       _Diskqueue_KeepAlive_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "diskqueue.proto",
}

func init() { proto.RegisterFile("diskqueue.proto", fileDescriptor_diskqueue_3971262521e2d53f) }

var fileDescriptor_diskqueue_3971262521e2d53f = []byte{
	// 392 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x53, 0x51, 0x4b, 0xc2, 0x50,
	0x18, 0x75, 0xce, 0xd4, 0x7d, 0xea, 0x94, 0x2f, 0xb3, 0xb9, 0x7a, 0x90, 0xfb, 0xb4, 0x87, 0x90,
	0x50, 0x08, 0x8a, 0x20, 0x2c, 0x23, 0x44, 0x04, 0x59, 0xbf, 0x60, 0x9b, 0x17, 0x1b, 0x6e, 0x39,
	0xbd, 0x5b, 0xe0, 0x6f, 0xec, 0x4f, 0xc5, 0x76, 0xdd, 0xdc, 0xd6, 0xe8, 0xa5, 0x37, 0xef, 0xf9,
	0xce, 0xf9, 0xbe, 0xe3, 0x39, 0x0c, 0xda, 0x2b, 0x9b, 0x6d, 0x76, 0x01, 0x0d, 0xe8, 0xd0, 0xdb,
	0x6f, 0xfd, 0x2d, 0x4a, 0x09, 0x40, 0xe6, 0x50, 0x5b, 0x50, 0xc6, 0x8c, 0x35, 0x45, 0x19, 0xca,
	0xb3, 0xa9, 0x22, 0x0c, 0x04, 0xad, 0xa2, 0x97, 0x67, 0x53, 0x44, 0xa8, 0xac, 0x0c, 0xdf, 0x50,
	0xca, 0x03, 0x41, 0x6b, 0xea, 0xd1, 0x6f, 0xbc, 0x06, 0xc9, 0xb7, 0x5d, 0xca, 0x7c, 0xc3, 0xf5,
	0x14, 0x71, 0x20, 0x68, 0xa2, 0x7e, 0x02, 0xc8, 0x03, 0xc8, 0xcb, 0xc0, 0x74, 0x6c, 0xf6, 0xa1,
	0xd3, 0x5d, 0x40, 0x99, 0x8f, 0x5d, 0x38, 0xf3, 0xb7, 0x9e, 0x6d, 0x45, 0x6b, 0x25, 0x9d, 0x3f,
	0x8a, 0x36, 0x13, 0x19, 0x9a, 0x89, 0xd6, 0x73, 0x0e, 0x44, 0x83, 0xce, 0x7b, 0x60, 0x32, 0x6b,
	0x6f, 0x9b, 0xf4, 0xcf, 0x6d, 0xa4, 0x03, 0x72, 0x8a, 0x19, 0x6a, 0xdb, 0xd0, 0x7a, 0x31, 0x3e,
	0x2d, 0xea, 0x1c, 0x85, 0xa4, 0x05, 0x8d, 0x18, 0x08, 0xe7, 0x2d, 0x68, 0x2c, 0x03, 0x27, 0x99,
	0xde, 0x83, 0xc4, 0x9f, 0x9e, 0x73, 0xc0, 0x1b, 0xa8, 0xb9, 0x3c, 0x90, 0xe8, 0x4a, 0x63, 0x84,
	0xc3, 0x53, 0x7c, 0xc7, 0xa8, 0xf4, 0x98, 0x42, 0x10, 0x3a, 0x73, 0x4a, 0xbd, 0x89, 0x63, 0x7f,
	0xc5, 0x2e, 0x43, 0x3f, 0x29, 0x2c, 0xbc, 0x47, 0x00, 0x26, 0xd6, 0x26, 0xf5, 0x2f, 0x5c, 0xb6,
	0x4e, 0xa2, 0xe6, 0x0f, 0x02, 0x50, 0x8f, 0x38, 0x9e, 0x73, 0x18, 0x7d, 0x8b, 0x20, 0x4d, 0xe3,
	0xa3, 0xf8, 0x04, 0xb5, 0x63, 0x32, 0xd8, 0x4f, 0x79, 0xc9, 0x26, 0xad, 0x5e, 0x16, 0x8d, 0xc2,
	0xe3, 0x25, 0x7c, 0x86, 0xfa, 0xe2, 0x5f, 0x1b, 0x34, 0x01, 0x5f, 0x41, 0x4a, 0x42, 0xc6, 0xab,
	0x14, 0x33, 0x5f, 0x92, 0xda, 0x2f, 0x1e, 0x72, 0x2b, 0x8f, 0x50, 0xe5, 0x45, 0xa0, 0x92, 0xa2,
	0x65, 0xca, 0x52, 0x7b, 0x05, 0x13, 0xae, 0xbe, 0x83, 0x4a, 0x58, 0x14, 0xf6, 0x32, 0x4e, 0x93,
	0x22, 0xd5, 0xee, 0x2f, 0x9c, 0xeb, 0xde, 0x40, 0x4a, 0x1a, 0xc9, 0x98, 0xcf, 0x77, 0x97, 0x31,
	0x9f, 0x2b, 0xb1, 0x74, 0x2b, 0xe0, 0x18, 0xc4, 0x89, 0xb5, 0xc1, 0x8b, 0x14, 0xeb, 0x54, 0xac,
	0x7a, 0x9e, 0x87, 0x23, 0x99, 0x59, 0x8d, 0x3e, 0xba, 0xf1, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0xa7, 0xfa, 0x5b, 0x26, 0x87, 0x03, 0x00, 0x00,
}