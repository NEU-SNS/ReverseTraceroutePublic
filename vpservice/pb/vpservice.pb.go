// Code generated by protoc-gen-go. DO NOT EDIT.
// source: vpservice/pb/vpservice.proto

package pb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type VantagePoint struct {
	Ip                   uint32   `protobuf:"varint,1,opt,name=ip,proto3" json:"ip,omitempty"`
	Hostname             string   `protobuf:"bytes,2,opt,name=hostname,proto3" json:"hostname,omitempty"`
	Site                 string   `protobuf:"bytes,3,opt,name=site,proto3" json:"site,omitempty"`
	Timestamp            bool     `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	RecordRoute          bool     `protobuf:"varint,5,opt,name=record_route,json=recordRoute,proto3" json:"record_route,omitempty"`
	Spoof                bool     `protobuf:"varint,6,opt,name=spoof,proto3" json:"spoof,omitempty"`
	RecSpoof             bool     `protobuf:"varint,7,opt,name=rec_spoof,json=recSpoof,proto3" json:"rec_spoof,omitempty"`
	Ping                 bool     `protobuf:"varint,8,opt,name=ping,proto3" json:"ping,omitempty"`
	Trace                bool     `protobuf:"varint,9,opt,name=trace,proto3" json:"trace,omitempty"`
	// XXX_NoUnkeyedLiteral struct{} `json:"-"`
	// XXX_unrecognized     []byte   `json:"-"`
	// XXX_sizecache        int32    `json:"-"`
}

func (m *VantagePoint) Reset()         { *m = VantagePoint{} }
func (m *VantagePoint) String() string { return proto.CompactTextString(m) }
func (*VantagePoint) ProtoMessage()    {}
func (*VantagePoint) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{0}
}

func (m *VantagePoint) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VantagePoint.Unmarshal(m, b)
}
func (m *VantagePoint) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VantagePoint.Marshal(b, m, deterministic)
}
func (m *VantagePoint) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VantagePoint.Merge(m, src)
}
func (m *VantagePoint) XXX_Size() int {
	return xxx_messageInfo_VantagePoint.Size(m)
}
func (m *VantagePoint) XXX_DiscardUnknown() {
	xxx_messageInfo_VantagePoint.DiscardUnknown(m)
}

var xxx_messageInfo_VantagePoint proto.InternalMessageInfo

func (m *VantagePoint) GetIp() uint32 {
	if m != nil {
		return m.Ip
	}
	return 0
}

func (m *VantagePoint) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *VantagePoint) GetSite() string {
	if m != nil {
		return m.Site
	}
	return ""
}

func (m *VantagePoint) GetTimestamp() bool {
	if m != nil {
		return m.Timestamp
	}
	return false
}

func (m *VantagePoint) GetRecordRoute() bool {
	if m != nil {
		return m.RecordRoute
	}
	return false
}

func (m *VantagePoint) GetSpoof() bool {
	if m != nil {
		return m.Spoof
	}
	return false
}

func (m *VantagePoint) GetRecSpoof() bool {
	if m != nil {
		return m.RecSpoof
	}
	return false
}

func (m *VantagePoint) GetPing() bool {
	if m != nil {
		return m.Ping
	}
	return false
}

func (m *VantagePoint) GetTrace() bool {
	if m != nil {
		return m.Trace
	}
	return false
}

type VPRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VPRequest) Reset()         { *m = VPRequest{} }
func (m *VPRequest) String() string { return proto.CompactTextString(m) }
func (*VPRequest) ProtoMessage()    {}
func (*VPRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{1}
}

func (m *VPRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VPRequest.Unmarshal(m, b)
}
func (m *VPRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VPRequest.Marshal(b, m, deterministic)
}
func (m *VPRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VPRequest.Merge(m, src)
}
func (m *VPRequest) XXX_Size() int {
	return xxx_messageInfo_VPRequest.Size(m)
}
func (m *VPRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_VPRequest.DiscardUnknown(m)
}

var xxx_messageInfo_VPRequest proto.InternalMessageInfo

type VPReturn struct {
	Vps                  []*VantagePoint `protobuf:"bytes,1,rep,name=vps,proto3" json:"vps,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *VPReturn) Reset()         { *m = VPReturn{} }
func (m *VPReturn) String() string { return proto.CompactTextString(m) }
func (*VPReturn) ProtoMessage()    {}
func (*VPReturn) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{2}
}

func (m *VPReturn) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VPReturn.Unmarshal(m, b)
}
func (m *VPReturn) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VPReturn.Marshal(b, m, deterministic)
}
func (m *VPReturn) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VPReturn.Merge(m, src)
}
func (m *VPReturn) XXX_Size() int {
	return xxx_messageInfo_VPReturn.Size(m)
}
func (m *VPReturn) XXX_DiscardUnknown() {
	xxx_messageInfo_VPReturn.DiscardUnknown(m)
}

var xxx_messageInfo_VPReturn proto.InternalMessageInfo

func (m *VPReturn) GetVps() []*VantagePoint {
	if m != nil {
		return m.Vps
	}
	return nil
}

type RRSpooferRequest struct {
	Addr                 uint32   `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	Max                  uint32   `protobuf:"varint,2,opt,name=max,proto3" json:"max,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RRSpooferRequest) Reset()         { *m = RRSpooferRequest{} }
func (m *RRSpooferRequest) String() string { return proto.CompactTextString(m) }
func (*RRSpooferRequest) ProtoMessage()    {}
func (*RRSpooferRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{3}
}

func (m *RRSpooferRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RRSpooferRequest.Unmarshal(m, b)
}
func (m *RRSpooferRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RRSpooferRequest.Marshal(b, m, deterministic)
}
func (m *RRSpooferRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RRSpooferRequest.Merge(m, src)
}
func (m *RRSpooferRequest) XXX_Size() int {
	return xxx_messageInfo_RRSpooferRequest.Size(m)
}
func (m *RRSpooferRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RRSpooferRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RRSpooferRequest proto.InternalMessageInfo

func (m *RRSpooferRequest) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *RRSpooferRequest) GetMax() uint32 {
	if m != nil {
		return m.Max
	}
	return 0
}

type RRSpooferResponse struct {
	Addr                 uint32          `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	Max                  uint32          `protobuf:"varint,2,opt,name=max,proto3" json:"max,omitempty"`
	Spoofers             []*VantagePoint `protobuf:"bytes,3,rep,name=spoofers,proto3" json:"spoofers,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *RRSpooferResponse) Reset()         { *m = RRSpooferResponse{} }
func (m *RRSpooferResponse) String() string { return proto.CompactTextString(m) }
func (*RRSpooferResponse) ProtoMessage()    {}
func (*RRSpooferResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{4}
}

func (m *RRSpooferResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RRSpooferResponse.Unmarshal(m, b)
}
func (m *RRSpooferResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RRSpooferResponse.Marshal(b, m, deterministic)
}
func (m *RRSpooferResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RRSpooferResponse.Merge(m, src)
}
func (m *RRSpooferResponse) XXX_Size() int {
	return xxx_messageInfo_RRSpooferResponse.Size(m)
}
func (m *RRSpooferResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RRSpooferResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RRSpooferResponse proto.InternalMessageInfo

func (m *RRSpooferResponse) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *RRSpooferResponse) GetMax() uint32 {
	if m != nil {
		return m.Max
	}
	return 0
}

func (m *RRSpooferResponse) GetSpoofers() []*VantagePoint {
	if m != nil {
		return m.Spoofers
	}
	return nil
}

type TSSpooferRequest struct {
	Addr                 uint32   `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	Max                  uint32   `protobuf:"varint,2,opt,name=max,proto3" json:"max,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TSSpooferRequest) Reset()         { *m = TSSpooferRequest{} }
func (m *TSSpooferRequest) String() string { return proto.CompactTextString(m) }
func (*TSSpooferRequest) ProtoMessage()    {}
func (*TSSpooferRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{5}
}

func (m *TSSpooferRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TSSpooferRequest.Unmarshal(m, b)
}
func (m *TSSpooferRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TSSpooferRequest.Marshal(b, m, deterministic)
}
func (m *TSSpooferRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TSSpooferRequest.Merge(m, src)
}
func (m *TSSpooferRequest) XXX_Size() int {
	return xxx_messageInfo_TSSpooferRequest.Size(m)
}
func (m *TSSpooferRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_TSSpooferRequest.DiscardUnknown(m)
}

var xxx_messageInfo_TSSpooferRequest proto.InternalMessageInfo

func (m *TSSpooferRequest) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *TSSpooferRequest) GetMax() uint32 {
	if m != nil {
		return m.Max
	}
	return 0
}

type TSSpooferResponse struct {
	Addr                 uint32          `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	Max                  uint32          `protobuf:"varint,2,opt,name=max,proto3" json:"max,omitempty"`
	Spoofers             []*VantagePoint `protobuf:"bytes,3,rep,name=spoofers,proto3" json:"spoofers,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *TSSpooferResponse) Reset()         { *m = TSSpooferResponse{} }
func (m *TSSpooferResponse) String() string { return proto.CompactTextString(m) }
func (*TSSpooferResponse) ProtoMessage()    {}
func (*TSSpooferResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{6}
}

func (m *TSSpooferResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TSSpooferResponse.Unmarshal(m, b)
}
func (m *TSSpooferResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TSSpooferResponse.Marshal(b, m, deterministic)
}
func (m *TSSpooferResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TSSpooferResponse.Merge(m, src)
}
func (m *TSSpooferResponse) XXX_Size() int {
	return xxx_messageInfo_TSSpooferResponse.Size(m)
}
func (m *TSSpooferResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_TSSpooferResponse.DiscardUnknown(m)
}

var xxx_messageInfo_TSSpooferResponse proto.InternalMessageInfo

func (m *TSSpooferResponse) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *TSSpooferResponse) GetMax() uint32 {
	if m != nil {
		return m.Max
	}
	return 0
}

func (m *TSSpooferResponse) GetSpoofers() []*VantagePoint {
	if m != nil {
		return m.Spoofers
	}
	return nil
}

type TestNewVPCapabilitiesRequest struct {
	Addr                 uint32   `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	Hostname             string   `protobuf:"bytes,2,opt,name=hostname,proto3" json:"hostname,omitempty"`
	Site                 string   `protobuf:"bytes,3,opt,name=site,proto3" json:"site,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TestNewVPCapabilitiesRequest) Reset()         { *m = TestNewVPCapabilitiesRequest{} }
func (m *TestNewVPCapabilitiesRequest) String() string { return proto.CompactTextString(m) }
func (*TestNewVPCapabilitiesRequest) ProtoMessage()    {}
func (*TestNewVPCapabilitiesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{7}
}

func (m *TestNewVPCapabilitiesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestNewVPCapabilitiesRequest.Unmarshal(m, b)
}
func (m *TestNewVPCapabilitiesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestNewVPCapabilitiesRequest.Marshal(b, m, deterministic)
}
func (m *TestNewVPCapabilitiesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestNewVPCapabilitiesRequest.Merge(m, src)
}
func (m *TestNewVPCapabilitiesRequest) XXX_Size() int {
	return xxx_messageInfo_TestNewVPCapabilitiesRequest.Size(m)
}
func (m *TestNewVPCapabilitiesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_TestNewVPCapabilitiesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_TestNewVPCapabilitiesRequest proto.InternalMessageInfo

func (m *TestNewVPCapabilitiesRequest) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *TestNewVPCapabilitiesRequest) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *TestNewVPCapabilitiesRequest) GetSite() string {
	if m != nil {
		return m.Site
	}
	return ""
}

type TestNewVPCapabilitiesResponse struct {
	CanReverseTraceroutes bool     `protobuf:"varint,1,opt,name=can_reverse_traceroutes,json=canReverseTraceroutes,proto3" json:"can_reverse_traceroutes,omitempty"`
	XXX_NoUnkeyedLiteral  struct{} `json:"-"`
	XXX_unrecognized      []byte   `json:"-"`
	XXX_sizecache         int32    `json:"-"`
}

func (m *TestNewVPCapabilitiesResponse) Reset()         { *m = TestNewVPCapabilitiesResponse{} }
func (m *TestNewVPCapabilitiesResponse) String() string { return proto.CompactTextString(m) }
func (*TestNewVPCapabilitiesResponse) ProtoMessage()    {}
func (*TestNewVPCapabilitiesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{8}
}

func (m *TestNewVPCapabilitiesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestNewVPCapabilitiesResponse.Unmarshal(m, b)
}
func (m *TestNewVPCapabilitiesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestNewVPCapabilitiesResponse.Marshal(b, m, deterministic)
}
func (m *TestNewVPCapabilitiesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestNewVPCapabilitiesResponse.Merge(m, src)
}
func (m *TestNewVPCapabilitiesResponse) XXX_Size() int {
	return xxx_messageInfo_TestNewVPCapabilitiesResponse.Size(m)
}
func (m *TestNewVPCapabilitiesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_TestNewVPCapabilitiesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_TestNewVPCapabilitiesResponse proto.InternalMessageInfo

func (m *TestNewVPCapabilitiesResponse) GetCanReverseTraceroutes() bool {
	if m != nil {
		return m.CanReverseTraceroutes
	}
	return false
}

type TryUnquarantineRequest struct {
	VantagePoints           []*VantagePoint `protobuf:"bytes,1,rep,name=vantage_points,json=vantagePoints,proto3" json:"vantage_points,omitempty"`
	VantagePointsForTesting []*VantagePoint `protobuf:"bytes,2,rep,name=vantage_points_for_testing,json=vantagePointsForTesting,proto3" json:"vantage_points_for_testing,omitempty"`
	IsTryAllVantagePoints   bool            `protobuf:"varint,3,opt,name=is_try_all_vantage_points,json=isTryAllVantagePoints,proto3" json:"is_try_all_vantage_points,omitempty"`
	IsSelfForTesting        bool            `protobuf:"varint,4,opt,name=is_self_for_testing,json=isSelfForTesting,proto3" json:"is_self_for_testing,omitempty"`
	IsTestOnlyActive        bool            `protobuf:"varint,5,opt,name=is_test_only_active,json=isTestOnlyActive,proto3" json:"is_test_only_active,omitempty"`
	XXX_NoUnkeyedLiteral    struct{}        `json:"-"`
	XXX_unrecognized        []byte          `json:"-"`
	XXX_sizecache           int32           `json:"-"`
}

func (m *TryUnquarantineRequest) Reset()         { *m = TryUnquarantineRequest{} }
func (m *TryUnquarantineRequest) String() string { return proto.CompactTextString(m) }
func (*TryUnquarantineRequest) ProtoMessage()    {}
func (*TryUnquarantineRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{9}
}

func (m *TryUnquarantineRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TryUnquarantineRequest.Unmarshal(m, b)
}
func (m *TryUnquarantineRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TryUnquarantineRequest.Marshal(b, m, deterministic)
}
func (m *TryUnquarantineRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TryUnquarantineRequest.Merge(m, src)
}
func (m *TryUnquarantineRequest) XXX_Size() int {
	return xxx_messageInfo_TryUnquarantineRequest.Size(m)
}
func (m *TryUnquarantineRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_TryUnquarantineRequest.DiscardUnknown(m)
}

var xxx_messageInfo_TryUnquarantineRequest proto.InternalMessageInfo

func (m *TryUnquarantineRequest) GetVantagePoints() []*VantagePoint {
	if m != nil {
		return m.VantagePoints
	}
	return nil
}

func (m *TryUnquarantineRequest) GetVantagePointsForTesting() []*VantagePoint {
	if m != nil {
		return m.VantagePointsForTesting
	}
	return nil
}

func (m *TryUnquarantineRequest) GetIsTryAllVantagePoints() bool {
	if m != nil {
		return m.IsTryAllVantagePoints
	}
	return false
}

func (m *TryUnquarantineRequest) GetIsSelfForTesting() bool {
	if m != nil {
		return m.IsSelfForTesting
	}
	return false
}

func (m *TryUnquarantineRequest) GetIsTestOnlyActive() bool {
	if m != nil {
		return m.IsTestOnlyActive
	}
	return false
}

type TryUnquarantineResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TryUnquarantineResponse) Reset()         { *m = TryUnquarantineResponse{} }
func (m *TryUnquarantineResponse) String() string { return proto.CompactTextString(m) }
func (*TryUnquarantineResponse) ProtoMessage()    {}
func (*TryUnquarantineResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e212206844e12f66, []int{10}
}

func (m *TryUnquarantineResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TryUnquarantineResponse.Unmarshal(m, b)
}
func (m *TryUnquarantineResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TryUnquarantineResponse.Marshal(b, m, deterministic)
}
func (m *TryUnquarantineResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TryUnquarantineResponse.Merge(m, src)
}
func (m *TryUnquarantineResponse) XXX_Size() int {
	return xxx_messageInfo_TryUnquarantineResponse.Size(m)
}
func (m *TryUnquarantineResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_TryUnquarantineResponse.DiscardUnknown(m)
}

var xxx_messageInfo_TryUnquarantineResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*VantagePoint)(nil), "pb.VantagePoint")
	proto.RegisterType((*VPRequest)(nil), "pb.VPRequest")
	proto.RegisterType((*VPReturn)(nil), "pb.VPReturn")
	proto.RegisterType((*RRSpooferRequest)(nil), "pb.RRSpooferRequest")
	proto.RegisterType((*RRSpooferResponse)(nil), "pb.RRSpooferResponse")
	proto.RegisterType((*TSSpooferRequest)(nil), "pb.TSSpooferRequest")
	proto.RegisterType((*TSSpooferResponse)(nil), "pb.TSSpooferResponse")
	proto.RegisterType((*TestNewVPCapabilitiesRequest)(nil), "pb.TestNewVPCapabilitiesRequest")
	proto.RegisterType((*TestNewVPCapabilitiesResponse)(nil), "pb.TestNewVPCapabilitiesResponse")
	proto.RegisterType((*TryUnquarantineRequest)(nil), "pb.TryUnquarantineRequest")
	proto.RegisterType((*TryUnquarantineResponse)(nil), "pb.TryUnquarantineResponse")
}

func init() {
	proto.RegisterFile("vpservice/pb/vpservice.proto", fileDescriptor_e212206844e12f66)
}

var fileDescriptor_e212206844e12f66 = []byte{
	// 634 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x54, 0xcb, 0x6e, 0xd3, 0x4c,
	0x14, 0x6e, 0x9c, 0xb6, 0xbf, 0x73, 0xda, 0xf4, 0x0f, 0x43, 0x4b, 0x5d, 0xb7, 0x48, 0xa9, 0x37,
	0x64, 0x01, 0xa9, 0x54, 0x24, 0xe8, 0x0a, 0xa9, 0x42, 0xa2, 0x1b, 0x2e, 0x91, 0x13, 0xc2, 0x0e,
	0x6b, 0xe2, 0x9e, 0x94, 0x91, 0x9c, 0xf1, 0x74, 0x66, 0x62, 0xc8, 0x9b, 0xf1, 0x3c, 0xac, 0x79,
	0x08, 0x34, 0x33, 0x89, 0x9b, 0xa4, 0x4d, 0x05, 0x48, 0xec, 0xe6, 0x5c, 0xbe, 0x73, 0xf3, 0xf7,
	0x19, 0x8e, 0x0a, 0xa1, 0x50, 0x16, 0x2c, 0xc5, 0x13, 0x31, 0x38, 0x29, 0x8d, 0xb6, 0x90, 0xb9,
	0xce, 0x89, 0x27, 0x06, 0xd1, 0xcf, 0x0a, 0x6c, 0xf7, 0x29, 0xd7, 0xf4, 0x0a, 0x3b, 0x39, 0xe3,
	0x9a, 0xec, 0x80, 0xc7, 0x44, 0x50, 0x69, 0x56, 0x5a, 0xf5, 0xd8, 0x63, 0x82, 0x84, 0xe0, 0x7f,
	0xc9, 0x95, 0xe6, 0x74, 0x84, 0x81, 0xd7, 0xac, 0xb4, 0x6a, 0x71, 0x69, 0x13, 0x02, 0xeb, 0x8a,
	0x69, 0x0c, 0xaa, 0xd6, 0x6f, 0xdf, 0xe4, 0x08, 0x6a, 0x9a, 0x8d, 0x50, 0x69, 0x3a, 0x12, 0xc1,
	0x7a, 0xb3, 0xd2, 0xf2, 0xe3, 0x1b, 0x07, 0x39, 0x86, 0x6d, 0x89, 0x69, 0x2e, 0x2f, 0x13, 0x99,
	0x8f, 0x35, 0x06, 0x1b, 0x36, 0x61, 0xcb, 0xf9, 0x62, 0xe3, 0x22, 0xbb, 0xb0, 0xa1, 0x44, 0x9e,
	0x0f, 0x83, 0x4d, 0x1b, 0x73, 0x06, 0x39, 0x84, 0x9a, 0xc4, 0x34, 0x71, 0x91, 0xff, 0x6c, 0xc4,
	0x97, 0x98, 0x76, 0x6d, 0x90, 0xc0, 0xba, 0x60, 0xfc, 0x2a, 0xf0, 0xad, 0xdf, 0xbe, 0x4d, 0x19,
	0x2d, 0x69, 0x8a, 0x41, 0xcd, 0x95, 0xb1, 0x46, 0xb4, 0x05, 0xb5, 0x7e, 0x27, 0xc6, 0xeb, 0x31,
	0x2a, 0x1d, 0xb5, 0xc1, 0x37, 0x86, 0x1e, 0x4b, 0x4e, 0x22, 0xa8, 0x16, 0x42, 0x05, 0x95, 0x66,
	0xb5, 0xb5, 0x75, 0xda, 0x68, 0x8b, 0x41, 0x7b, 0xfe, 0x2a, 0xb1, 0x09, 0x46, 0x67, 0xd0, 0x88,
	0x63, 0xdb, 0x11, 0xe5, 0xb4, 0x86, 0x69, 0x4d, 0x2f, 0x2f, 0xe5, 0xf4, 0x60, 0xf6, 0x4d, 0x1a,
	0x50, 0x1d, 0xd1, 0x6f, 0xf6, 0x5a, 0xf5, 0xd8, 0x3c, 0xa3, 0x2b, 0x78, 0x30, 0x87, 0x54, 0x22,
	0xe7, 0x0a, 0x7f, 0x0f, 0x4a, 0x9e, 0x82, 0xaf, 0x1c, 0x50, 0x05, 0xd5, 0x15, 0xd3, 0x95, 0x19,
	0x66, 0xc4, 0x5e, 0xf7, 0x6f, 0x47, 0x9c, 0x43, 0xfe, 0xc3, 0x11, 0x07, 0x70, 0xd4, 0x43, 0xa5,
	0xdf, 0xe3, 0xd7, 0x7e, 0xe7, 0x35, 0x15, 0x74, 0xc0, 0x32, 0xa6, 0x19, 0xaa, 0xfb, 0xc6, 0xfd,
	0x43, 0x12, 0x46, 0x9f, 0xe0, 0xf1, 0x8a, 0x1e, 0xd3, 0xc5, 0x5e, 0xc0, 0x7e, 0x4a, 0x79, 0x22,
	0xb1, 0x40, 0xa9, 0x30, 0xb1, 0xe4, 0xb0, 0x8c, 0x54, 0xb6, 0xaf, 0x1f, 0xef, 0xa5, 0x94, 0xc7,
	0x2e, 0xda, 0xbb, 0x09, 0x46, 0xdf, 0x3d, 0x78, 0xd4, 0x93, 0x93, 0x8f, 0xfc, 0x7a, 0x4c, 0x25,
	0xe5, 0x9a, 0x71, 0x9c, 0xcd, 0xfd, 0x12, 0x76, 0x0a, 0xb7, 0x71, 0x22, 0xcc, 0xca, 0xab, 0xc9,
	0x54, 0x2f, 0xe6, 0x2c, 0x45, 0xde, 0x41, 0xb8, 0x08, 0x4c, 0x86, 0xb9, 0x4c, 0x34, 0x2a, 0x6d,
	0x38, 0xed, 0xad, 0x28, 0xb2, 0xbf, 0x50, 0xe4, 0x4d, 0x2e, 0x7b, 0x0e, 0x40, 0xce, 0xe0, 0x80,
	0xa9, 0x44, 0xcb, 0x49, 0x42, 0xb3, 0x2c, 0x59, 0x1a, 0xa9, 0xea, 0x96, 0x63, 0xaa, 0x27, 0x27,
	0xe7, 0x59, 0xd6, 0x5f, 0x18, 0xe4, 0x19, 0x3c, 0x64, 0x2a, 0x51, 0x98, 0x0d, 0x17, 0x26, 0x70,
	0x22, 0x6e, 0x30, 0xd5, 0xc5, 0x6c, 0x38, 0xd7, 0xc8, 0xa5, 0x9b, 0xac, 0x24, 0xe7, 0xd9, 0x24,
	0xa1, 0xa9, 0x66, 0xc5, 0x4c, 0xd2, 0x0d, 0xa6, 0x4c, 0xde, 0x07, 0x9e, 0x4d, 0xce, 0xad, 0x3f,
	0x3a, 0x80, 0xfd, 0x5b, 0x97, 0x73, 0x5f, 0xe3, 0xf4, 0x87, 0x67, 0x64, 0xd9, 0x75, 0x3f, 0x27,
	0xf2, 0x04, 0x36, 0x2f, 0x50, 0xf7, 0x3b, 0x8a, 0xd4, 0xed, 0xd6, 0x33, 0xbd, 0x86, 0xdb, 0x33,
	0xd3, 0x28, 0x36, 0x5a, 0x23, 0xaf, 0xa0, 0x7e, 0x81, 0xba, 0x14, 0x96, 0x22, 0xbb, 0x26, 0x61,
	0x59, 0xa2, 0xe1, 0xde, 0x92, 0xd7, 0x35, 0x2d, 0xf1, 0x25, 0xeb, 0xa7, 0xf8, 0x65, 0xfd, 0x38,
	0xfc, 0x2d, 0x6d, 0x44, 0x6b, 0xe4, 0x33, 0xec, 0xdd, 0xc9, 0x32, 0xd2, 0xb4, 0x88, 0x7b, 0x48,
	0x1e, 0x1e, 0xdf, 0x93, 0x51, 0xd6, 0x7f, 0x0b, 0xff, 0x2f, 0x5d, 0x8c, 0x84, 0x16, 0x77, 0x27,
	0x01, 0xc3, 0xc3, 0x3b, 0x63, 0xb3, 0x6a, 0x83, 0x4d, 0xfb, 0xd3, 0x7f, 0xfe, 0x2b, 0x00, 0x00,
	0xff, 0xff, 0x06, 0x3a, 0x4f, 0x9d, 0x14, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// VPServiceClient is the client API for VPService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type VPServiceClient interface {
	GetVPs(ctx context.Context, in *VPRequest, opts ...grpc.CallOption) (*VPReturn, error)
	GetRRSpoofers(ctx context.Context, in *RRSpooferRequest, opts ...grpc.CallOption) (*RRSpooferResponse, error)
	GetTSSpoofers(ctx context.Context, in *TSSpooferRequest, opts ...grpc.CallOption) (*TSSpooferResponse, error)
	TestNewVPCapabilities(ctx context.Context, in *TestNewVPCapabilitiesRequest, opts ...grpc.CallOption) (*TestNewVPCapabilitiesResponse, error)
	TryUnquarantine(ctx context.Context, in *TryUnquarantineRequest, opts ...grpc.CallOption) (*TryUnquarantineResponse, error)
}

type vPServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewVPServiceClient(cc grpc.ClientConnInterface) VPServiceClient {
	return &vPServiceClient{cc}
}

func (c *vPServiceClient) GetVPs(ctx context.Context, in *VPRequest, opts ...grpc.CallOption) (*VPReturn, error) {
	out := new(VPReturn)
	err := c.cc.Invoke(ctx, "/pb.VPService/GetVPs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vPServiceClient) GetRRSpoofers(ctx context.Context, in *RRSpooferRequest, opts ...grpc.CallOption) (*RRSpooferResponse, error) {
	out := new(RRSpooferResponse)
	err := c.cc.Invoke(ctx, "/pb.VPService/GetRRSpoofers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vPServiceClient) GetTSSpoofers(ctx context.Context, in *TSSpooferRequest, opts ...grpc.CallOption) (*TSSpooferResponse, error) {
	out := new(TSSpooferResponse)
	err := c.cc.Invoke(ctx, "/pb.VPService/GetTSSpoofers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vPServiceClient) TestNewVPCapabilities(ctx context.Context, in *TestNewVPCapabilitiesRequest, opts ...grpc.CallOption) (*TestNewVPCapabilitiesResponse, error) {
	out := new(TestNewVPCapabilitiesResponse)
	err := c.cc.Invoke(ctx, "/pb.VPService/TestNewVPCapabilities", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vPServiceClient) TryUnquarantine(ctx context.Context, in *TryUnquarantineRequest, opts ...grpc.CallOption) (*TryUnquarantineResponse, error) {
	out := new(TryUnquarantineResponse)
	err := c.cc.Invoke(ctx, "/pb.VPService/TryUnquarantine", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// VPServiceServer is the server API for VPService service.
type VPServiceServer interface {
	GetVPs(context.Context, *VPRequest) (*VPReturn, error)
	GetRRSpoofers(context.Context, *RRSpooferRequest) (*RRSpooferResponse, error)
	GetTSSpoofers(context.Context, *TSSpooferRequest) (*TSSpooferResponse, error)
	TestNewVPCapabilities(context.Context, *TestNewVPCapabilitiesRequest) (*TestNewVPCapabilitiesResponse, error)
	TryUnquarantine(context.Context, *TryUnquarantineRequest) (*TryUnquarantineResponse, error)
}

// UnimplementedVPServiceServer can be embedded to have forward compatible implementations.
type UnimplementedVPServiceServer struct {
}

func (*UnimplementedVPServiceServer) GetVPs(ctx context.Context, req *VPRequest) (*VPReturn, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVPs not implemented")
}
func (*UnimplementedVPServiceServer) GetRRSpoofers(ctx context.Context, req *RRSpooferRequest) (*RRSpooferResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRRSpoofers not implemented")
}
func (*UnimplementedVPServiceServer) GetTSSpoofers(ctx context.Context, req *TSSpooferRequest) (*TSSpooferResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTSSpoofers not implemented")
}
func (*UnimplementedVPServiceServer) TestNewVPCapabilities(ctx context.Context, req *TestNewVPCapabilitiesRequest) (*TestNewVPCapabilitiesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TestNewVPCapabilities not implemented")
}
func (*UnimplementedVPServiceServer) TryUnquarantine(ctx context.Context, req *TryUnquarantineRequest) (*TryUnquarantineResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TryUnquarantine not implemented")
}

func RegisterVPServiceServer(s *grpc.Server, srv VPServiceServer) {
	s.RegisterService(&_VPService_serviceDesc, srv)
}

func _VPService_GetVPs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VPRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VPServiceServer).GetVPs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.VPService/GetVPs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VPServiceServer).GetVPs(ctx, req.(*VPRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VPService_GetRRSpoofers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RRSpooferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VPServiceServer).GetRRSpoofers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.VPService/GetRRSpoofers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VPServiceServer).GetRRSpoofers(ctx, req.(*RRSpooferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VPService_GetTSSpoofers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TSSpooferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VPServiceServer).GetTSSpoofers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.VPService/GetTSSpoofers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VPServiceServer).GetTSSpoofers(ctx, req.(*TSSpooferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VPService_TestNewVPCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TestNewVPCapabilitiesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VPServiceServer).TestNewVPCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.VPService/TestNewVPCapabilities",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VPServiceServer).TestNewVPCapabilities(ctx, req.(*TestNewVPCapabilitiesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VPService_TryUnquarantine_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TryUnquarantineRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VPServiceServer).TryUnquarantine(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.VPService/TryUnquarantine",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VPServiceServer).TryUnquarantine(ctx, req.(*TryUnquarantineRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _VPService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.VPService",
	HandlerType: (*VPServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetVPs",
			Handler:    _VPService_GetVPs_Handler,
		},
		{
			MethodName: "GetRRSpoofers",
			Handler:    _VPService_GetRRSpoofers_Handler,
		},
		{
			MethodName: "GetTSSpoofers",
			Handler:    _VPService_GetTSSpoofers_Handler,
		},
		{
			MethodName: "TestNewVPCapabilities",
			Handler:    _VPService_TestNewVPCapabilities_Handler,
		},
		{
			MethodName: "TryUnquarantine",
			Handler:    _VPService_TryUnquarantine_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "vpservice/pb/vpservice.proto",
}