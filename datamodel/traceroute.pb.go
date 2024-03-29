// Code generated by protoc-gen-go. DO NOT EDIT.
// source: traceroute.proto

package datamodel

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type TracerouteMeasurement struct {
	Staleness            int64    `protobuf:"varint,1,opt,name=staleness,proto3" json:"staleness,omitempty"`
	Dst                  uint32   `protobuf:"varint,3,opt,name=dst,proto3" json:"dst,omitempty"`
	Confidence           string   `protobuf:"bytes,4,opt,name=confidence,proto3" json:"confidence,omitempty"`
	Dport                string   `protobuf:"bytes,5,opt,name=dport,proto3" json:"dport,omitempty"`
	FirstHop             string   `protobuf:"bytes,6,opt,name=first_hop,json=firstHop,proto3" json:"first_hop,omitempty"`
	GapLimit             string   `protobuf:"bytes,7,opt,name=gap_limit,json=gapLimit,proto3" json:"gap_limit,omitempty"`
	GapAction            string   `protobuf:"bytes,8,opt,name=gap_action,json=gapAction,proto3" json:"gap_action,omitempty"`
	MaxTtl               string   `protobuf:"bytes,9,opt,name=max_ttl,json=maxTtl,proto3" json:"max_ttl,omitempty"`
	PathDiscov           bool     `protobuf:"varint,10,opt,name=path_discov,json=pathDiscov,proto3" json:"path_discov,omitempty"`
	Loops                string   `protobuf:"bytes,11,opt,name=loops,proto3" json:"loops,omitempty"`
	LoopAction           string   `protobuf:"bytes,12,opt,name=loop_action,json=loopAction,proto3" json:"loop_action,omitempty"`
	Payload              string   `protobuf:"bytes,13,opt,name=payload,proto3" json:"payload,omitempty"`
	Method               string   `protobuf:"bytes,14,opt,name=method,proto3" json:"method,omitempty"`
	Attempts             string   `protobuf:"bytes,15,opt,name=attempts,proto3" json:"attempts,omitempty"`
	SendAll              bool     `protobuf:"varint,16,opt,name=send_all,json=sendAll,proto3" json:"send_all,omitempty"`
	Sport                string   `protobuf:"bytes,17,opt,name=sport,proto3" json:"sport,omitempty"`
	Src                  uint32   `protobuf:"varint,18,opt,name=src,proto3" json:"src,omitempty"`
	Tos                  string   `protobuf:"bytes,19,opt,name=tos,proto3" json:"tos,omitempty"`
	TimeExceeded         bool     `protobuf:"varint,20,opt,name=time_exceeded,json=timeExceeded,proto3" json:"time_exceeded,omitempty"`
	UserId               string   `protobuf:"bytes,21,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Wait                 string   `protobuf:"bytes,22,opt,name=wait,proto3" json:"wait,omitempty"`
	WaitProbe            string   `protobuf:"bytes,23,opt,name=wait_probe,json=waitProbe,proto3" json:"wait_probe,omitempty"`
	GssEntry             string   `protobuf:"bytes,24,opt,name=gss_entry,json=gssEntry,proto3" json:"gss_entry,omitempty"`
	LssName              string   `protobuf:"bytes,25,opt,name=lss_name,json=lssName,proto3" json:"lss_name,omitempty"`
	Timeout              int64    `protobuf:"varint,26,opt,name=timeout,proto3" json:"timeout,omitempty"`
	CheckCache           bool     `protobuf:"varint,27,opt,name=check_cache,json=checkCache,proto3" json:"check_cache,omitempty"`
	CheckDb              bool     `protobuf:"varint,28,opt,name=check_db,json=checkDb,proto3" json:"check_db,omitempty"`
	IsRipeAtlas          bool     `protobuf:"varint,29,opt,name=is_ripe_atlas,json=isRipeAtlas,proto3" json:"is_ripe_atlas,omitempty"`
	RipeApiKey           string   `protobuf:"bytes,30,opt,name=ripe_api_key,json=ripeApiKey,proto3" json:"ripe_api_key,omitempty"`
	RipeQueryUrl         string   `protobuf:"bytes,31,opt,name=ripe_query_url,json=ripeQueryUrl,proto3" json:"ripe_query_url,omitempty"`
	RipeQueryBody        string   `protobuf:"bytes,32,opt,name=ripe_query_body,json=ripeQueryBody,proto3" json:"ripe_query_body,omitempty"`
	NRipeProbeIds        int32    `protobuf:"varint,33,opt,name=n_ripe_probe_ids,json=nRipeProbeIds,proto3" json:"n_ripe_probe_ids,omitempty"`
	SaveDb               bool     `protobuf:"varint,34,opt,name=save_db,json=saveDb,proto3" json:"save_db,omitempty"`
	Label                string   `protobuf:"bytes,35,opt,name=label,proto3" json:"label,omitempty"`
	FromRevtr            bool     `protobuf:"varint,36,opt,name=from_revtr,json=fromRevtr,proto3" json:"from_revtr,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TracerouteMeasurement) Reset()         { *m = TracerouteMeasurement{} }
func (m *TracerouteMeasurement) String() string { return proto.CompactTextString(m) }
func (*TracerouteMeasurement) ProtoMessage()    {}
func (*TracerouteMeasurement) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{0}
}

func (m *TracerouteMeasurement) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TracerouteMeasurement.Unmarshal(m, b)
}
func (m *TracerouteMeasurement) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TracerouteMeasurement.Marshal(b, m, deterministic)
}
func (m *TracerouteMeasurement) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TracerouteMeasurement.Merge(m, src)
}
func (m *TracerouteMeasurement) XXX_Size() int {
	return xxx_messageInfo_TracerouteMeasurement.Size(m)
}
func (m *TracerouteMeasurement) XXX_DiscardUnknown() {
	xxx_messageInfo_TracerouteMeasurement.DiscardUnknown(m)
}

var xxx_messageInfo_TracerouteMeasurement proto.InternalMessageInfo

func (m *TracerouteMeasurement) GetStaleness() int64 {
	if m != nil {
		return m.Staleness
	}
	return 0
}

func (m *TracerouteMeasurement) GetDst() uint32 {
	if m != nil {
		return m.Dst
	}
	return 0
}

func (m *TracerouteMeasurement) GetConfidence() string {
	if m != nil {
		return m.Confidence
	}
	return ""
}

func (m *TracerouteMeasurement) GetDport() string {
	if m != nil {
		return m.Dport
	}
	return ""
}

func (m *TracerouteMeasurement) GetFirstHop() string {
	if m != nil {
		return m.FirstHop
	}
	return ""
}

func (m *TracerouteMeasurement) GetGapLimit() string {
	if m != nil {
		return m.GapLimit
	}
	return ""
}

func (m *TracerouteMeasurement) GetGapAction() string {
	if m != nil {
		return m.GapAction
	}
	return ""
}

func (m *TracerouteMeasurement) GetMaxTtl() string {
	if m != nil {
		return m.MaxTtl
	}
	return ""
}

func (m *TracerouteMeasurement) GetPathDiscov() bool {
	if m != nil {
		return m.PathDiscov
	}
	return false
}

func (m *TracerouteMeasurement) GetLoops() string {
	if m != nil {
		return m.Loops
	}
	return ""
}

func (m *TracerouteMeasurement) GetLoopAction() string {
	if m != nil {
		return m.LoopAction
	}
	return ""
}

func (m *TracerouteMeasurement) GetPayload() string {
	if m != nil {
		return m.Payload
	}
	return ""
}

func (m *TracerouteMeasurement) GetMethod() string {
	if m != nil {
		return m.Method
	}
	return ""
}

func (m *TracerouteMeasurement) GetAttempts() string {
	if m != nil {
		return m.Attempts
	}
	return ""
}

func (m *TracerouteMeasurement) GetSendAll() bool {
	if m != nil {
		return m.SendAll
	}
	return false
}

func (m *TracerouteMeasurement) GetSport() string {
	if m != nil {
		return m.Sport
	}
	return ""
}

func (m *TracerouteMeasurement) GetSrc() uint32 {
	if m != nil {
		return m.Src
	}
	return 0
}

func (m *TracerouteMeasurement) GetTos() string {
	if m != nil {
		return m.Tos
	}
	return ""
}

func (m *TracerouteMeasurement) GetTimeExceeded() bool {
	if m != nil {
		return m.TimeExceeded
	}
	return false
}

func (m *TracerouteMeasurement) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *TracerouteMeasurement) GetWait() string {
	if m != nil {
		return m.Wait
	}
	return ""
}

func (m *TracerouteMeasurement) GetWaitProbe() string {
	if m != nil {
		return m.WaitProbe
	}
	return ""
}

func (m *TracerouteMeasurement) GetGssEntry() string {
	if m != nil {
		return m.GssEntry
	}
	return ""
}

func (m *TracerouteMeasurement) GetLssName() string {
	if m != nil {
		return m.LssName
	}
	return ""
}

func (m *TracerouteMeasurement) GetTimeout() int64 {
	if m != nil {
		return m.Timeout
	}
	return 0
}

func (m *TracerouteMeasurement) GetCheckCache() bool {
	if m != nil {
		return m.CheckCache
	}
	return false
}

func (m *TracerouteMeasurement) GetCheckDb() bool {
	if m != nil {
		return m.CheckDb
	}
	return false
}

func (m *TracerouteMeasurement) GetIsRipeAtlas() bool {
	if m != nil {
		return m.IsRipeAtlas
	}
	return false
}

func (m *TracerouteMeasurement) GetRipeApiKey() string {
	if m != nil {
		return m.RipeApiKey
	}
	return ""
}

func (m *TracerouteMeasurement) GetRipeQueryUrl() string {
	if m != nil {
		return m.RipeQueryUrl
	}
	return ""
}

func (m *TracerouteMeasurement) GetRipeQueryBody() string {
	if m != nil {
		return m.RipeQueryBody
	}
	return ""
}

func (m *TracerouteMeasurement) GetNRipeProbeIds() int32 {
	if m != nil {
		return m.NRipeProbeIds
	}
	return 0
}

func (m *TracerouteMeasurement) GetSaveDb() bool {
	if m != nil {
		return m.SaveDb
	}
	return false
}

func (m *TracerouteMeasurement) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

func (m *TracerouteMeasurement) GetFromRevtr() bool {
	if m != nil {
		return m.FromRevtr
	}
	return false
}

type TracerouteArg struct {
	Traceroutes          []*TracerouteMeasurement `protobuf:"bytes,1,rep,name=traceroutes,proto3" json:"traceroutes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *TracerouteArg) Reset()         { *m = TracerouteArg{} }
func (m *TracerouteArg) String() string { return proto.CompactTextString(m) }
func (*TracerouteArg) ProtoMessage()    {}
func (*TracerouteArg) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{1}
}

func (m *TracerouteArg) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TracerouteArg.Unmarshal(m, b)
}
func (m *TracerouteArg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TracerouteArg.Marshal(b, m, deterministic)
}
func (m *TracerouteArg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TracerouteArg.Merge(m, src)
}
func (m *TracerouteArg) XXX_Size() int {
	return xxx_messageInfo_TracerouteArg.Size(m)
}
func (m *TracerouteArg) XXX_DiscardUnknown() {
	xxx_messageInfo_TracerouteArg.DiscardUnknown(m)
}

var xxx_messageInfo_TracerouteArg proto.InternalMessageInfo

func (m *TracerouteArg) GetTraceroutes() []*TracerouteMeasurement {
	if m != nil {
		return m.Traceroutes
	}
	return nil
}

type TracerouteArgResp struct {
	Traceroutes          []*Traceroute `protobuf:"bytes,1,rep,name=traceroutes,proto3" json:"traceroutes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *TracerouteArgResp) Reset()         { *m = TracerouteArgResp{} }
func (m *TracerouteArgResp) String() string { return proto.CompactTextString(m) }
func (*TracerouteArgResp) ProtoMessage()    {}
func (*TracerouteArgResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{2}
}

func (m *TracerouteArgResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TracerouteArgResp.Unmarshal(m, b)
}
func (m *TracerouteArgResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TracerouteArgResp.Marshal(b, m, deterministic)
}
func (m *TracerouteArgResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TracerouteArgResp.Merge(m, src)
}
func (m *TracerouteArgResp) XXX_Size() int {
	return xxx_messageInfo_TracerouteArgResp.Size(m)
}
func (m *TracerouteArgResp) XXX_DiscardUnknown() {
	xxx_messageInfo_TracerouteArgResp.DiscardUnknown(m)
}

var xxx_messageInfo_TracerouteArgResp proto.InternalMessageInfo

func (m *TracerouteArgResp) GetTraceroutes() []*Traceroute {
	if m != nil {
		return m.Traceroutes
	}
	return nil
}

type TracerouteHop struct {
	Addr                 uint32          `protobuf:"varint,1,opt,name=addr,proto3" json:"addr,omitempty"`
	ProbeTtl             uint32          `protobuf:"varint,2,opt,name=probe_ttl,json=probeTtl,proto3" json:"probe_ttl,omitempty"`
	ProbeId              uint32          `protobuf:"varint,3,opt,name=probe_id,json=probeId,proto3" json:"probe_id,omitempty"`
	ProbeSize            uint32          `protobuf:"varint,4,opt,name=probe_size,json=probeSize,proto3" json:"probe_size,omitempty"`
	Rtt                  *RTT            `protobuf:"bytes,5,opt,name=rtt,proto3" json:"rtt,omitempty"`
	ReplyTtl             uint32          `protobuf:"varint,6,opt,name=reply_ttl,json=replyTtl,proto3" json:"reply_ttl,omitempty"`
	ReplyTos             uint32          `protobuf:"varint,7,opt,name=reply_tos,json=replyTos,proto3" json:"reply_tos,omitempty"`
	ReplySize            uint32          `protobuf:"varint,8,opt,name=reply_size,json=replySize,proto3" json:"reply_size,omitempty"`
	ReplyIpid            uint32          `protobuf:"varint,9,opt,name=reply_ipid,json=replyIpid,proto3" json:"reply_ipid,omitempty"`
	IcmpType             uint32          `protobuf:"varint,10,opt,name=icmp_type,json=icmpType,proto3" json:"icmp_type,omitempty"`
	IcmpCode             uint32          `protobuf:"varint,11,opt,name=icmp_code,json=icmpCode,proto3" json:"icmp_code,omitempty"`
	IcmpQTtl             uint32          `protobuf:"varint,12,opt,name=icmp_q_ttl,json=icmpQTtl,proto3" json:"icmp_q_ttl,omitempty"`
	IcmpQIpl             uint32          `protobuf:"varint,13,opt,name=icmp_q_ipl,json=icmpQIpl,proto3" json:"icmp_q_ipl,omitempty"`
	IcmpQTos             uint32          `protobuf:"varint,14,opt,name=icmp_q_tos,json=icmpQTos,proto3" json:"icmp_q_tos,omitempty"`
	IcmpExt              *ICMPExtensions `protobuf:"bytes,15,opt,name=icmp_ext,json=icmpExt,proto3" json:"icmp_ext,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *TracerouteHop) Reset()         { *m = TracerouteHop{} }
func (m *TracerouteHop) String() string { return proto.CompactTextString(m) }
func (*TracerouteHop) ProtoMessage()    {}
func (*TracerouteHop) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{3}
}

func (m *TracerouteHop) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TracerouteHop.Unmarshal(m, b)
}
func (m *TracerouteHop) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TracerouteHop.Marshal(b, m, deterministic)
}
func (m *TracerouteHop) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TracerouteHop.Merge(m, src)
}
func (m *TracerouteHop) XXX_Size() int {
	return xxx_messageInfo_TracerouteHop.Size(m)
}
func (m *TracerouteHop) XXX_DiscardUnknown() {
	xxx_messageInfo_TracerouteHop.DiscardUnknown(m)
}

var xxx_messageInfo_TracerouteHop proto.InternalMessageInfo

func (m *TracerouteHop) GetAddr() uint32 {
	if m != nil {
		return m.Addr
	}
	return 0
}

func (m *TracerouteHop) GetProbeTtl() uint32 {
	if m != nil {
		return m.ProbeTtl
	}
	return 0
}

func (m *TracerouteHop) GetProbeId() uint32 {
	if m != nil {
		return m.ProbeId
	}
	return 0
}

func (m *TracerouteHop) GetProbeSize() uint32 {
	if m != nil {
		return m.ProbeSize
	}
	return 0
}

func (m *TracerouteHop) GetRtt() *RTT {
	if m != nil {
		return m.Rtt
	}
	return nil
}

func (m *TracerouteHop) GetReplyTtl() uint32 {
	if m != nil {
		return m.ReplyTtl
	}
	return 0
}

func (m *TracerouteHop) GetReplyTos() uint32 {
	if m != nil {
		return m.ReplyTos
	}
	return 0
}

func (m *TracerouteHop) GetReplySize() uint32 {
	if m != nil {
		return m.ReplySize
	}
	return 0
}

func (m *TracerouteHop) GetReplyIpid() uint32 {
	if m != nil {
		return m.ReplyIpid
	}
	return 0
}

func (m *TracerouteHop) GetIcmpType() uint32 {
	if m != nil {
		return m.IcmpType
	}
	return 0
}

func (m *TracerouteHop) GetIcmpCode() uint32 {
	if m != nil {
		return m.IcmpCode
	}
	return 0
}

func (m *TracerouteHop) GetIcmpQTtl() uint32 {
	if m != nil {
		return m.IcmpQTtl
	}
	return 0
}

func (m *TracerouteHop) GetIcmpQIpl() uint32 {
	if m != nil {
		return m.IcmpQIpl
	}
	return 0
}

func (m *TracerouteHop) GetIcmpQTos() uint32 {
	if m != nil {
		return m.IcmpQTos
	}
	return 0
}

func (m *TracerouteHop) GetIcmpExt() *ICMPExtensions {
	if m != nil {
		return m.IcmpExt
	}
	return nil
}

type ICMPExtensions struct {
	Length               uint32           `protobuf:"varint,1,opt,name=length,proto3" json:"length,omitempty"`
	IcmpExtensionList    []*ICMPExtension `protobuf:"bytes,2,rep,name=icmp_extension_list,json=icmpExtensionList,proto3" json:"icmp_extension_list,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *ICMPExtensions) Reset()         { *m = ICMPExtensions{} }
func (m *ICMPExtensions) String() string { return proto.CompactTextString(m) }
func (*ICMPExtensions) ProtoMessage()    {}
func (*ICMPExtensions) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{4}
}

func (m *ICMPExtensions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ICMPExtensions.Unmarshal(m, b)
}
func (m *ICMPExtensions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ICMPExtensions.Marshal(b, m, deterministic)
}
func (m *ICMPExtensions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ICMPExtensions.Merge(m, src)
}
func (m *ICMPExtensions) XXX_Size() int {
	return xxx_messageInfo_ICMPExtensions.Size(m)
}
func (m *ICMPExtensions) XXX_DiscardUnknown() {
	xxx_messageInfo_ICMPExtensions.DiscardUnknown(m)
}

var xxx_messageInfo_ICMPExtensions proto.InternalMessageInfo

func (m *ICMPExtensions) GetLength() uint32 {
	if m != nil {
		return m.Length
	}
	return 0
}

func (m *ICMPExtensions) GetIcmpExtensionList() []*ICMPExtension {
	if m != nil {
		return m.IcmpExtensionList
	}
	return nil
}

type ICMPExtension struct {
	Length               uint32   `protobuf:"varint,1,opt,name=length,proto3" json:"length,omitempty"`
	ClassNumber          uint32   `protobuf:"varint,2,opt,name=class_number,json=classNumber,proto3" json:"class_number,omitempty"`
	TypeNumber           uint32   `protobuf:"varint,3,opt,name=type_number,json=typeNumber,proto3" json:"type_number,omitempty"`
	Data                 []byte   `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ICMPExtension) Reset()         { *m = ICMPExtension{} }
func (m *ICMPExtension) String() string { return proto.CompactTextString(m) }
func (*ICMPExtension) ProtoMessage()    {}
func (*ICMPExtension) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{5}
}

func (m *ICMPExtension) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ICMPExtension.Unmarshal(m, b)
}
func (m *ICMPExtension) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ICMPExtension.Marshal(b, m, deterministic)
}
func (m *ICMPExtension) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ICMPExtension.Merge(m, src)
}
func (m *ICMPExtension) XXX_Size() int {
	return xxx_messageInfo_ICMPExtension.Size(m)
}
func (m *ICMPExtension) XXX_DiscardUnknown() {
	xxx_messageInfo_ICMPExtension.DiscardUnknown(m)
}

var xxx_messageInfo_ICMPExtension proto.InternalMessageInfo

func (m *ICMPExtension) GetLength() uint32 {
	if m != nil {
		return m.Length
	}
	return 0
}

func (m *ICMPExtension) GetClassNumber() uint32 {
	if m != nil {
		return m.ClassNumber
	}
	return 0
}

func (m *ICMPExtension) GetTypeNumber() uint32 {
	if m != nil {
		return m.TypeNumber
	}
	return 0
}

func (m *ICMPExtension) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type Traceroute struct {
	Type                 string           `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	UserId               uint32           `protobuf:"varint,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Method               string           `protobuf:"bytes,3,opt,name=method,proto3" json:"method,omitempty"`
	Src                  uint32           `protobuf:"varint,4,opt,name=src,proto3" json:"src,omitempty"`
	Dst                  uint32           `protobuf:"varint,5,opt,name=dst,proto3" json:"dst,omitempty"`
	Sport                uint32           `protobuf:"varint,6,opt,name=sport,proto3" json:"sport,omitempty"`
	Dport                uint32           `protobuf:"varint,7,opt,name=dport,proto3" json:"dport,omitempty"`
	StopReason           string           `protobuf:"bytes,8,opt,name=stop_reason,json=stopReason,proto3" json:"stop_reason,omitempty"`
	StopData             uint32           `protobuf:"varint,9,opt,name=stop_data,json=stopData,proto3" json:"stop_data,omitempty"`
	Start                *TracerouteTime  `protobuf:"bytes,10,opt,name=start,proto3" json:"start,omitempty"`
	HopCount             uint32           `protobuf:"varint,11,opt,name=hop_count,json=hopCount,proto3" json:"hop_count,omitempty"`
	Attempts             uint32           `protobuf:"varint,12,opt,name=attempts,proto3" json:"attempts,omitempty"`
	Hoplimit             uint32           `protobuf:"varint,13,opt,name=hoplimit,proto3" json:"hoplimit,omitempty"`
	Firsthop             uint32           `protobuf:"varint,14,opt,name=firsthop,proto3" json:"firsthop,omitempty"`
	Wait                 uint32           `protobuf:"varint,15,opt,name=wait,proto3" json:"wait,omitempty"`
	WaitProbe            uint32           `protobuf:"varint,16,opt,name=wait_probe,json=waitProbe,proto3" json:"wait_probe,omitempty"`
	Tos                  uint32           `protobuf:"varint,17,opt,name=tos,proto3" json:"tos,omitempty"`
	ProbeSize            uint32           `protobuf:"varint,18,opt,name=probe_size,json=probeSize,proto3" json:"probe_size,omitempty"`
	Hops                 []*TracerouteHop `protobuf:"bytes,19,rep,name=hops,proto3" json:"hops,omitempty"`
	Error                string           `protobuf:"bytes,20,opt,name=error,proto3" json:"error,omitempty"`
	Version              string           `protobuf:"bytes,21,opt,name=version,proto3" json:"version,omitempty"`
	GapLimit             uint32           `protobuf:"varint,22,opt,name=gap_limit,json=gapLimit,proto3" json:"gap_limit,omitempty"`
	Id                   int64            `protobuf:"varint,23,opt,name=id,proto3" json:"id,omitempty"`
	SourceAsn            int32            `protobuf:"varint,24,opt,name=source_asn,json=sourceAsn,proto3" json:"source_asn,omitempty"`
	Platform             string           `protobuf:"bytes,25,opt,name=platform,proto3" json:"platform,omitempty"`
	FromCache            bool             `protobuf:"varint,26,opt,name=from_cache,json=fromCache,proto3" json:"from_cache,omitempty"`
	Label                string           `protobuf:"bytes,27,opt,name=label,proto3" json:"label,omitempty"`
	FromRevtr            bool             `protobuf:"varint,28,opt,name=from_revtr,json=fromRevtr,proto3" json:"from_revtr,omitempty"`
	RevtrMeasurementId   uint64           `protobuf:"varint,29,opt,name=revtr_measurement_id,json=revtrMeasurementId,proto3" json:"revtr_measurement_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *Traceroute) Reset()         { *m = Traceroute{} }
func (m *Traceroute) String() string { return proto.CompactTextString(m) }
func (*Traceroute) ProtoMessage()    {}
func (*Traceroute) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{6}
}

func (m *Traceroute) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Traceroute.Unmarshal(m, b)
}
func (m *Traceroute) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Traceroute.Marshal(b, m, deterministic)
}
func (m *Traceroute) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Traceroute.Merge(m, src)
}
func (m *Traceroute) XXX_Size() int {
	return xxx_messageInfo_Traceroute.Size(m)
}
func (m *Traceroute) XXX_DiscardUnknown() {
	xxx_messageInfo_Traceroute.DiscardUnknown(m)
}

var xxx_messageInfo_Traceroute proto.InternalMessageInfo

func (m *Traceroute) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *Traceroute) GetUserId() uint32 {
	if m != nil {
		return m.UserId
	}
	return 0
}

func (m *Traceroute) GetMethod() string {
	if m != nil {
		return m.Method
	}
	return ""
}

func (m *Traceroute) GetSrc() uint32 {
	if m != nil {
		return m.Src
	}
	return 0
}

func (m *Traceroute) GetDst() uint32 {
	if m != nil {
		return m.Dst
	}
	return 0
}

func (m *Traceroute) GetSport() uint32 {
	if m != nil {
		return m.Sport
	}
	return 0
}

func (m *Traceroute) GetDport() uint32 {
	if m != nil {
		return m.Dport
	}
	return 0
}

func (m *Traceroute) GetStopReason() string {
	if m != nil {
		return m.StopReason
	}
	return ""
}

func (m *Traceroute) GetStopData() uint32 {
	if m != nil {
		return m.StopData
	}
	return 0
}

func (m *Traceroute) GetStart() *TracerouteTime {
	if m != nil {
		return m.Start
	}
	return nil
}

func (m *Traceroute) GetHopCount() uint32 {
	if m != nil {
		return m.HopCount
	}
	return 0
}

func (m *Traceroute) GetAttempts() uint32 {
	if m != nil {
		return m.Attempts
	}
	return 0
}

func (m *Traceroute) GetHoplimit() uint32 {
	if m != nil {
		return m.Hoplimit
	}
	return 0
}

func (m *Traceroute) GetFirsthop() uint32 {
	if m != nil {
		return m.Firsthop
	}
	return 0
}

func (m *Traceroute) GetWait() uint32 {
	if m != nil {
		return m.Wait
	}
	return 0
}

func (m *Traceroute) GetWaitProbe() uint32 {
	if m != nil {
		return m.WaitProbe
	}
	return 0
}

func (m *Traceroute) GetTos() uint32 {
	if m != nil {
		return m.Tos
	}
	return 0
}

func (m *Traceroute) GetProbeSize() uint32 {
	if m != nil {
		return m.ProbeSize
	}
	return 0
}

func (m *Traceroute) GetHops() []*TracerouteHop {
	if m != nil {
		return m.Hops
	}
	return nil
}

func (m *Traceroute) GetError() string {
	if m != nil {
		return m.Error
	}
	return ""
}

func (m *Traceroute) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func (m *Traceroute) GetGapLimit() uint32 {
	if m != nil {
		return m.GapLimit
	}
	return 0
}

func (m *Traceroute) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Traceroute) GetSourceAsn() int32 {
	if m != nil {
		return m.SourceAsn
	}
	return 0
}

func (m *Traceroute) GetPlatform() string {
	if m != nil {
		return m.Platform
	}
	return ""
}

func (m *Traceroute) GetFromCache() bool {
	if m != nil {
		return m.FromCache
	}
	return false
}

func (m *Traceroute) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

func (m *Traceroute) GetFromRevtr() bool {
	if m != nil {
		return m.FromRevtr
	}
	return false
}

func (m *Traceroute) GetRevtrMeasurementId() uint64 {
	if m != nil {
		return m.RevtrMeasurementId
	}
	return 0
}

type TracerouteTime struct {
	Sec                  int64    `protobuf:"varint,1,opt,name=sec,proto3" json:"sec,omitempty"`
	Usec                 int64    `protobuf:"varint,2,opt,name=usec,proto3" json:"usec,omitempty"`
	Ftime                string   `protobuf:"bytes,3,opt,name=ftime,proto3" json:"ftime,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TracerouteTime) Reset()         { *m = TracerouteTime{} }
func (m *TracerouteTime) String() string { return proto.CompactTextString(m) }
func (*TracerouteTime) ProtoMessage()    {}
func (*TracerouteTime) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{7}
}

func (m *TracerouteTime) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TracerouteTime.Unmarshal(m, b)
}
func (m *TracerouteTime) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TracerouteTime.Marshal(b, m, deterministic)
}
func (m *TracerouteTime) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TracerouteTime.Merge(m, src)
}
func (m *TracerouteTime) XXX_Size() int {
	return xxx_messageInfo_TracerouteTime.Size(m)
}
func (m *TracerouteTime) XXX_DiscardUnknown() {
	xxx_messageInfo_TracerouteTime.DiscardUnknown(m)
}

var xxx_messageInfo_TracerouteTime proto.InternalMessageInfo

func (m *TracerouteTime) GetSec() int64 {
	if m != nil {
		return m.Sec
	}
	return 0
}

func (m *TracerouteTime) GetUsec() int64 {
	if m != nil {
		return m.Usec
	}
	return 0
}

func (m *TracerouteTime) GetFtime() string {
	if m != nil {
		return m.Ftime
	}
	return ""
}

type RIPEAtlasTracerouteArg struct {
	Query                string   `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RIPEAtlasTracerouteArg) Reset()         { *m = RIPEAtlasTracerouteArg{} }
func (m *RIPEAtlasTracerouteArg) String() string { return proto.CompactTextString(m) }
func (*RIPEAtlasTracerouteArg) ProtoMessage()    {}
func (*RIPEAtlasTracerouteArg) Descriptor() ([]byte, []int) {
	return fileDescriptor_e239eee109287058, []int{8}
}

func (m *RIPEAtlasTracerouteArg) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RIPEAtlasTracerouteArg.Unmarshal(m, b)
}
func (m *RIPEAtlasTracerouteArg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RIPEAtlasTracerouteArg.Marshal(b, m, deterministic)
}
func (m *RIPEAtlasTracerouteArg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RIPEAtlasTracerouteArg.Merge(m, src)
}
func (m *RIPEAtlasTracerouteArg) XXX_Size() int {
	return xxx_messageInfo_RIPEAtlasTracerouteArg.Size(m)
}
func (m *RIPEAtlasTracerouteArg) XXX_DiscardUnknown() {
	xxx_messageInfo_RIPEAtlasTracerouteArg.DiscardUnknown(m)
}

var xxx_messageInfo_RIPEAtlasTracerouteArg proto.InternalMessageInfo

func (m *RIPEAtlasTracerouteArg) GetQuery() string {
	if m != nil {
		return m.Query
	}
	return ""
}

func init() {
	proto.RegisterType((*TracerouteMeasurement)(nil), "datamodel.TracerouteMeasurement")
	proto.RegisterType((*TracerouteArg)(nil), "datamodel.TracerouteArg")
	proto.RegisterType((*TracerouteArgResp)(nil), "datamodel.TracerouteArgResp")
	proto.RegisterType((*TracerouteHop)(nil), "datamodel.TracerouteHop")
	proto.RegisterType((*ICMPExtensions)(nil), "datamodel.ICMPExtensions")
	proto.RegisterType((*ICMPExtension)(nil), "datamodel.ICMPExtension")
	proto.RegisterType((*Traceroute)(nil), "datamodel.Traceroute")
	proto.RegisterType((*TracerouteTime)(nil), "datamodel.TracerouteTime")
	proto.RegisterType((*RIPEAtlasTracerouteArg)(nil), "datamodel.RIPEAtlasTracerouteArg")
}

func init() {
	proto.RegisterFile("traceroute.proto", fileDescriptor_e239eee109287058)
}

var fileDescriptor_e239eee109287058 = []byte{
	// 1337 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x56, 0xdd, 0x6e, 0x1b, 0xb7,
	0x12, 0x86, 0xac, 0xff, 0x91, 0xa5, 0xd8, 0x8c, 0xe3, 0xd0, 0x8e, 0x93, 0x28, 0x4a, 0x70, 0x8e,
	0x2f, 0x0e, 0x7c, 0x8a, 0xb4, 0x40, 0xaf, 0x1d, 0xdb, 0x40, 0x84, 0x3a, 0x41, 0xb2, 0x71, 0x6f,
	0x7a, 0xb3, 0xa0, 0x76, 0x69, 0x8b, 0xc8, 0xee, 0x92, 0x21, 0x29, 0xd7, 0xca, 0x4d, 0x9f, 0xa9,
	0x2f, 0xd4, 0xa7, 0xe8, 0x03, 0x14, 0x33, 0x5c, 0xed, 0x4a, 0xa9, 0x73, 0xb5, 0x9c, 0xef, 0x9b,
	0x25, 0x87, 0x9c, 0xe1, 0x7c, 0x84, 0x1d, 0x6f, 0x45, 0x22, 0xad, 0x5e, 0x78, 0x79, 0x62, 0xac,
	0xf6, 0x9a, 0xf5, 0x53, 0xe1, 0x45, 0xae, 0x53, 0x99, 0x1d, 0x82, 0x57, 0x79, 0x09, 0x4f, 0xfe,
	0xee, 0xc2, 0xa3, 0xab, 0xca, 0xf7, 0x9d, 0x14, 0x6e, 0x61, 0x65, 0x2e, 0x0b, 0xcf, 0x8e, 0xa0,
	0xef, 0xbc, 0xc8, 0x64, 0x21, 0x9d, 0xe3, 0x8d, 0x71, 0xe3, 0xb8, 0x19, 0xd5, 0x00, 0xdb, 0x81,
	0x66, 0xea, 0x3c, 0x6f, 0x8e, 0x1b, 0xc7, 0xc3, 0x08, 0x87, 0xec, 0x19, 0x40, 0xa2, 0x8b, 0x6b,
	0x95, 0xca, 0x22, 0x91, 0xbc, 0x35, 0x6e, 0x1c, 0xf7, 0xa3, 0x35, 0x84, 0xed, 0x41, 0x3b, 0x35,
	0xda, 0x7a, 0xde, 0x26, 0x2a, 0x18, 0xec, 0x09, 0xf4, 0xaf, 0x95, 0x75, 0x3e, 0x9e, 0x6b, 0xc3,
	0x3b, 0xc4, 0xf4, 0x08, 0x78, 0xab, 0x0d, 0x92, 0x37, 0xc2, 0xc4, 0x99, 0xca, 0x95, 0xe7, 0xdd,
	0x40, 0xde, 0x08, 0x73, 0x89, 0x36, 0x7b, 0x0a, 0x80, 0xa4, 0x48, 0xbc, 0xd2, 0x05, 0xef, 0x11,
	0x8b, 0xee, 0xa7, 0x04, 0xb0, 0xc7, 0xd0, 0xcd, 0xc5, 0x5d, 0xec, 0x7d, 0xc6, 0xfb, 0xc4, 0x75,
	0x72, 0x71, 0x77, 0xe5, 0x33, 0xf6, 0x1c, 0x06, 0x46, 0xf8, 0x79, 0x9c, 0x2a, 0x97, 0xe8, 0x5b,
	0x0e, 0xe3, 0xc6, 0x71, 0x2f, 0x02, 0x84, 0xce, 0x09, 0xc1, 0x40, 0x33, 0xad, 0x8d, 0xe3, 0x83,
	0x10, 0x28, 0x19, 0xf8, 0x1b, 0x0e, 0x56, 0xeb, 0x6d, 0x87, 0xfd, 0x21, 0x54, 0x2e, 0xc8, 0xa1,
	0x6b, 0xc4, 0x32, 0xd3, 0x22, 0xe5, 0x43, 0x22, 0x57, 0x26, 0xdb, 0x87, 0x4e, 0x2e, 0xfd, 0x5c,
	0xa7, 0x7c, 0x54, 0x46, 0x42, 0x16, 0x3b, 0x84, 0x9e, 0xf0, 0x5e, 0xe6, 0xc6, 0x3b, 0xfe, 0x20,
	0xec, 0x6e, 0x65, 0xb3, 0x03, 0xe8, 0x39, 0x59, 0xa4, 0xb1, 0xc8, 0x32, 0xbe, 0x43, 0x21, 0x76,
	0xd1, 0x3e, 0xcd, 0x32, 0x8c, 0xcf, 0xd1, 0x41, 0xee, 0x86, 0xf8, 0xc8, 0xc0, 0x84, 0x38, 0x9b,
	0x70, 0x16, 0x12, 0xe2, 0x6c, 0x82, 0x88, 0xd7, 0x8e, 0x3f, 0x24, 0x2f, 0x1c, 0xb2, 0x97, 0x30,
	0xc4, 0xd4, 0xc7, 0xf2, 0x2e, 0x91, 0x32, 0x95, 0x29, 0xdf, 0xa3, 0x99, 0xb7, 0x11, 0xbc, 0x28,
	0x31, 0x3c, 0xb8, 0x85, 0x93, 0x36, 0x56, 0x29, 0x7f, 0x14, 0xc2, 0x45, 0x73, 0x9a, 0x32, 0x06,
	0xad, 0xdf, 0x85, 0xf2, 0x7c, 0x9f, 0x50, 0x1a, 0x63, 0x12, 0xf0, 0x1b, 0x1b, 0xab, 0x67, 0x92,
	0x3f, 0x0e, 0x49, 0x40, 0xe4, 0x03, 0x02, 0x94, 0x40, 0xe7, 0x62, 0x59, 0x78, 0xbb, 0xe4, 0xbc,
	0x4c, 0xa0, 0x73, 0x17, 0x68, 0xe3, 0x16, 0x33, 0xe7, 0xe2, 0x42, 0xe4, 0x92, 0x1f, 0x84, 0x13,
	0xcb, 0x9c, 0x7b, 0x2f, 0x72, 0x89, 0x67, 0x89, 0x31, 0xe9, 0x85, 0xe7, 0x87, 0x54, 0x79, 0x2b,
	0x13, 0xd3, 0x90, 0xcc, 0x65, 0xf2, 0x39, 0x4e, 0x44, 0x32, 0x97, 0xfc, 0x49, 0xc8, 0x1e, 0x41,
	0x67, 0x88, 0xe0, 0xac, 0xc1, 0x21, 0x9d, 0xf1, 0xa3, 0x70, 0x70, 0x64, 0x9f, 0xcf, 0xd8, 0x04,
	0x86, 0xca, 0xc5, 0x56, 0x19, 0x19, 0x0b, 0x9f, 0x09, 0xc7, 0x9f, 0x12, 0x3f, 0x50, 0x2e, 0x52,
	0x46, 0x9e, 0x22, 0xc4, 0xc6, 0xb0, 0x1d, 0x1c, 0x8c, 0x8a, 0x3f, 0xcb, 0x25, 0x7f, 0x16, 0xf2,
	0x8c, 0xd8, 0xa9, 0x51, 0xbf, 0xc8, 0x25, 0x7b, 0x05, 0x23, 0xf2, 0xf8, 0xb2, 0x90, 0x76, 0x19,
	0x2f, 0x6c, 0xc6, 0x9f, 0x93, 0x0f, 0xfd, 0xf7, 0x11, 0xc1, 0x5f, 0x6d, 0xc6, 0xfe, 0x03, 0x0f,
	0xd6, 0xbc, 0x66, 0x3a, 0x5d, 0xf2, 0x31, 0xb9, 0x0d, 0x2b, 0xb7, 0x37, 0x3a, 0x5d, 0xb2, 0xff,
	0xc2, 0x4e, 0x11, 0x42, 0xa2, 0x23, 0x8c, 0x55, 0xea, 0xf8, 0x8b, 0x71, 0xe3, 0xb8, 0x1d, 0x0d,
	0x0b, 0x8c, 0x8a, 0xce, 0x71, 0x9a, 0x3a, 0x4c, 0x8b, 0x13, 0xb7, 0x12, 0xb7, 0x35, 0xa1, 0xb0,
	0x3b, 0x68, 0x9e, 0xcf, 0xa8, 0x5c, 0xc5, 0x4c, 0x66, 0xfc, 0x65, 0x59, 0xae, 0x68, 0x60, 0x62,
	0xae, 0xad, 0xce, 0x63, 0x2b, 0x6f, 0xbd, 0xe5, 0xaf, 0xe8, 0x8f, 0x3e, 0x22, 0x11, 0x02, 0x93,
	0x4f, 0x30, 0xac, 0x6f, 0xfd, 0xa9, 0xbd, 0x61, 0x6f, 0x60, 0x50, 0xb7, 0x0c, 0xbc, 0xef, 0xcd,
	0xe3, 0xc1, 0xeb, 0xf1, 0x49, 0xd5, 0x34, 0x4e, 0xee, 0x6d, 0x12, 0xd1, 0xfa, 0x4f, 0x93, 0x4b,
	0xd8, 0xdd, 0x98, 0x34, 0x92, 0xce, 0xb0, 0x9f, 0xef, 0x9b, 0xf8, 0xd1, 0xbd, 0x13, 0x6f, 0xce,
	0xf6, 0x57, 0x73, 0x3d, 0x46, 0x6c, 0x07, 0x0c, 0x5a, 0x22, 0x4d, 0x2d, 0x35, 0xa3, 0x61, 0x44,
	0x63, 0xac, 0xb0, 0x70, 0x70, 0x78, 0xd1, 0xb7, 0x88, 0xe8, 0x11, 0x80, 0x57, 0xfd, 0x00, 0x7a,
	0xab, 0x53, 0x2d, 0x3b, 0x55, 0xd7, 0x84, 0xf3, 0xc4, 0xf3, 0x09, 0x94, 0x53, 0x5f, 0x43, 0xb7,
	0x1a, 0x46, 0x61, 0xa6, 0x4f, 0xea, 0xab, 0x64, 0x63, 0x68, 0x5a, 0x1f, 0x5a, 0xd5, 0xe0, 0xf5,
	0x68, 0x2d, 0xda, 0xe8, 0xea, 0x2a, 0x42, 0x0a, 0x17, 0xb6, 0xd2, 0x64, 0x4b, 0x5a, 0xb8, 0x13,
	0x16, 0x26, 0x00, 0x17, 0xae, 0x49, 0xed, 0xa8, 0x71, 0x55, 0xa4, 0x76, 0xb8, 0x74, 0x20, 0x69,
	0xe9, 0x5e, 0x58, 0x9a, 0x10, 0x5a, 0xba, 0xa2, 0x95, 0x51, 0x29, 0xf5, 0xae, 0x15, 0x3d, 0x35,
	0x2a, 0xc5, 0xa9, 0x55, 0x92, 0x9b, 0xd8, 0x2f, 0x8d, 0xa4, 0xe6, 0x35, 0x8c, 0x7a, 0x08, 0x5c,
	0x2d, 0x8d, 0xac, 0xc8, 0x44, 0xa7, 0x92, 0xda, 0x57, 0x49, 0x9e, 0xe9, 0x54, 0xb2, 0x23, 0x00,
	0x22, 0xbf, 0x50, 0xc8, 0xdb, 0x35, 0xfb, 0x11, 0x43, 0xae, 0x59, 0x65, 0x32, 0xea, 0x60, 0x2b,
	0x76, 0x6a, 0xd6, 0x59, 0xdc, 0xd1, 0x68, 0xfd, 0x5f, 0xed, 0xd8, 0x4f, 0x40, 0xe3, 0x58, 0xde,
	0x79, 0x6a, 0x64, 0x83, 0xd7, 0x07, 0x6b, 0x47, 0x36, 0x3d, 0x7b, 0xf7, 0xe1, 0xe2, 0xce, 0xcb,
	0xc2, 0x29, 0x5d, 0xb8, 0xa8, 0x8b, 0xae, 0x17, 0x77, 0x7e, 0x62, 0x61, 0xb4, 0x49, 0x61, 0xa3,
	0xcc, 0x64, 0x71, 0xe3, 0xe7, 0x65, 0x8a, 0x4b, 0x8b, 0xbd, 0x85, 0x87, 0xab, 0xf9, 0x83, 0x6b,
	0x9c, 0x29, 0xe7, 0xf9, 0x16, 0xd5, 0x12, 0xff, 0xde, 0x52, 0xd1, 0x6e, 0xb9, 0x52, 0x30, 0x2f,
	0x95, 0xf3, 0x93, 0x3f, 0x60, 0xb8, 0xe1, 0xf3, 0xdd, 0x25, 0x5f, 0xc0, 0x76, 0x92, 0x09, 0x6c,
	0x4f, 0x8b, 0x7c, 0x26, 0x6d, 0x59, 0x5a, 0x03, 0xc2, 0xde, 0x13, 0x84, 0xad, 0x08, 0x93, 0xb0,
	0xf2, 0x08, 0x05, 0x06, 0x08, 0x95, 0x0e, 0x0c, 0x5a, 0x18, 0x1a, 0x55, 0xd7, 0x76, 0x44, 0xe3,
	0xc9, 0x9f, 0x1d, 0x80, 0xba, 0xaa, 0xd1, 0x85, 0x12, 0xd9, 0x08, 0x3d, 0x15, 0xc7, 0xeb, 0x0d,
	0x38, 0xac, 0xba, 0x6a, 0xc0, 0xb5, 0x8e, 0x34, 0x37, 0x74, 0xa4, 0x6c, 0xfd, 0xad, 0x8d, 0xd6,
	0x8f, 0xea, 0xdc, 0xae, 0xd5, 0xb9, 0x12, 0x8d, 0x50, 0xaa, 0xa5, 0x68, 0x54, 0x9a, 0x1c, 0x6a,
	0xb4, 0xd4, 0xe4, 0xe7, 0x30, 0x70, 0x5e, 0x9b, 0xd8, 0x4a, 0xe1, 0x2a, 0x69, 0x05, 0x84, 0x22,
	0x42, 0xb0, 0xcc, 0xc8, 0x81, 0x76, 0x17, 0x2a, 0xb4, 0x87, 0xc0, 0xb9, 0xf0, 0x82, 0xfd, 0x1f,
	0xda, 0xce, 0x0b, 0xeb, 0xa9, 0x38, 0x37, 0x2b, 0xa1, 0xde, 0xf8, 0x95, 0xca, 0x65, 0x14, 0xfc,
	0x70, 0xb6, 0xb9, 0xc6, 0x9a, 0x5d, 0x14, 0x7e, 0x55, 0xb4, 0x73, 0x6d, 0xce, 0xd0, 0xde, 0xd0,
	0xc8, 0xb2, 0x64, 0x2b, 0x8d, 0x3c, 0x04, 0xf4, 0x0b, 0xaf, 0x83, 0x61, 0xf5, 0x1f, 0xd9, 0xc8,
	0xd1, 0x33, 0x02, 0x9f, 0x15, 0x65, 0xb9, 0xae, 0xec, 0x4a, 0xc8, 0x1e, 0x84, 0x3e, 0x72, 0x8f,
	0x90, 0xed, 0x84, 0x5b, 0x57, 0x0b, 0x59, 0xa9, 0xa5, 0xbb, 0xe1, 0x40, 0x7d, 0xb8, 0xc5, 0x6b,
	0x0d, 0x84, 0x7d, 0xdb, 0x40, 0xfe, 0x07, 0xad, 0x39, 0xbe, 0x21, 0x1e, 0xfe, 0xab, 0x46, 0x37,
	0x7a, 0x5a, 0x44, 0x5e, 0x98, 0x07, 0x69, 0xad, 0xb6, 0x24, 0xc8, 0xfd, 0x28, 0x18, 0xa8, 0x82,
	0xb7, 0xd2, 0x62, 0x99, 0x96, 0x4a, 0xbc, 0x32, 0x37, 0x1f, 0x46, 0xfb, 0x61, 0x7b, 0xd5, 0xc3,
	0x68, 0x04, 0x5b, 0x2a, 0x25, 0x2d, 0x6e, 0x46, 0x5b, 0x8a, 0x5a, 0x9d, 0xd3, 0x0b, 0x9b, 0xc8,
	0x58, 0xb8, 0x82, 0x54, 0xb8, 0x1d, 0xf5, 0x03, 0x72, 0xea, 0x0a, 0x3c, 0x29, 0x93, 0x09, 0x7f,
	0xad, 0x6d, 0x5e, 0xca, 0x70, 0x65, 0x57, 0x2a, 0x12, 0xc4, 0xf6, 0xb0, 0x56, 0x91, 0xa0, 0xb5,
	0x95, 0xf4, 0x3c, 0xf9, 0xbe, 0xf4, 0x1c, 0x7d, 0x23, 0x3d, 0xec, 0x07, 0xd8, 0x23, 0x26, 0xce,
	0x6b, 0x1d, 0xc1, 0x5a, 0x47, 0x31, 0x6e, 0x45, 0x8c, 0xb8, 0x35, 0x89, 0x99, 0xa6, 0x93, 0x4b,
	0x18, 0x6d, 0x56, 0x0e, 0x55, 0xbc, 0x4c, 0xca, 0x57, 0x29, 0x0e, 0x31, 0xa7, 0x0b, 0x84, 0xb6,
	0x08, 0xa2, 0x31, 0x86, 0x77, 0x8d, 0xef, 0x86, 0xf2, 0xba, 0x04, 0x63, 0x72, 0x02, 0xfb, 0xd1,
	0xf4, 0xc3, 0x05, 0xc9, 0xfd, 0xa6, 0x06, 0xee, 0x41, 0x9b, 0xe4, 0xba, 0xbc, 0x8d, 0xc1, 0x78,
	0x33, 0xf8, 0xad, 0x7e, 0x3a, 0xcf, 0x3a, 0xf4, 0x6a, 0xfe, 0xf1, 0x9f, 0x00, 0x00, 0x00, 0xff,
	0xff, 0x0e, 0xf6, 0xb8, 0x91, 0x60, 0x0b, 0x00, 0x00,
}
