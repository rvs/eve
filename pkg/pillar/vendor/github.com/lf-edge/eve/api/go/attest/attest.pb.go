// Code generated by protoc-gen-go. DO NOT EDIT.
// source: attest.proto

package attest

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

type ZAttestReqType int32

const (
	ZAttestReqType_ATTEST_REQ_NONE  ZAttestReqType = 0
	ZAttestReqType_ATTEST_REQ_CERT  ZAttestReqType = 1
	ZAttestReqType_ATTEST_REQ_NONCE ZAttestReqType = 2
	ZAttestReqType_ATTEST_REQ_QUOTE ZAttestReqType = 3
)

var ZAttestReqType_name = map[int32]string{
	0: "ATTEST_REQ_NONE",
	1: "ATTEST_REQ_CERT",
	2: "ATTEST_REQ_NONCE",
	3: "ATTEST_REQ_QUOTE",
}

var ZAttestReqType_value = map[string]int32{
	"ATTEST_REQ_NONE":  0,
	"ATTEST_REQ_CERT":  1,
	"ATTEST_REQ_NONCE": 2,
	"ATTEST_REQ_QUOTE": 3,
}

func (x ZAttestReqType) String() string {
	return proto.EnumName(ZAttestReqType_name, int32(x))
}

func (ZAttestReqType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{0}
}

type ZAttestRespType int32

const (
	ZAttestRespType_ATTEST_RESP_NONE       ZAttestRespType = 0
	ZAttestRespType_ATTEST_RESP_CERT       ZAttestRespType = 1
	ZAttestRespType_ATTEST_RESP_NONCE      ZAttestRespType = 2
	ZAttestRespType_ATTEST_RESP_QUOTE_RESP ZAttestRespType = 3
)

var ZAttestRespType_name = map[int32]string{
	0: "ATTEST_RESP_NONE",
	1: "ATTEST_RESP_CERT",
	2: "ATTEST_RESP_NONCE",
	3: "ATTEST_RESP_QUOTE_RESP",
}

var ZAttestRespType_value = map[string]int32{
	"ATTEST_RESP_NONE":       0,
	"ATTEST_RESP_CERT":       1,
	"ATTEST_RESP_NONCE":      2,
	"ATTEST_RESP_QUOTE_RESP": 3,
}

func (x ZAttestRespType) String() string {
	return proto.EnumName(ZAttestRespType_name, int32(x))
}

func (ZAttestRespType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{1}
}

type ZAttestResponseCode int32

const (
	ZAttestResponseCode_ATTEST_RESPONSE_NONE    ZAttestResponseCode = 0
	ZAttestResponseCode_ATTEST_RESPONSE_SUCCESS ZAttestResponseCode = 1
	ZAttestResponseCode_ATTEST_RESPONSE_FAILURE ZAttestResponseCode = 2
)

var ZAttestResponseCode_name = map[int32]string{
	0: "ATTEST_RESPONSE_NONE",
	1: "ATTEST_RESPONSE_SUCCESS",
	2: "ATTEST_RESPONSE_FAILURE",
}

var ZAttestResponseCode_value = map[string]int32{
	"ATTEST_RESPONSE_NONE":    0,
	"ATTEST_RESPONSE_SUCCESS": 1,
	"ATTEST_RESPONSE_FAILURE": 2,
}

func (x ZAttestResponseCode) String() string {
	return proto.EnumName(ZAttestResponseCode_name, int32(x))
}

func (ZAttestResponseCode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{2}
}

type ZEveCertHashType int32

const (
	ZEveCertHashType_HASH_NONE           ZEveCertHashType = 0
	ZEveCertHashType_HASH_SHA256_16bytes ZEveCertHashType = 1
)

var ZEveCertHashType_name = map[int32]string{
	0: "HASH_NONE",
	1: "HASH_SHA256_16bytes",
}

var ZEveCertHashType_value = map[string]int32{
	"HASH_NONE":           0,
	"HASH_SHA256_16bytes": 1,
}

func (x ZEveCertHashType) String() string {
	return proto.EnumName(ZEveCertHashType_name, int32(x))
}

func (ZEveCertHashType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{3}
}

type ZEveCertType int32

const (
	ZEveCertType_CERT_TYPE_DEVICE_NONE               ZEveCertType = 0
	ZEveCertType_CERT_TYPE_DEVICE_ONBOARDING         ZEveCertType = 1
	ZEveCertType_CERT_TYPE_DEVICE_RESTRICTED_SIGNING ZEveCertType = 2
	ZEveCertType_CERT_TYPE_DEVICE_ENDORSEMENT_RSA    ZEveCertType = 3
	ZEveCertType_CERT_TYPE_DEVICE_ECDH_EXCHANGE      ZEveCertType = 4
)

var ZEveCertType_name = map[int32]string{
	0: "CERT_TYPE_DEVICE_NONE",
	1: "CERT_TYPE_DEVICE_ONBOARDING",
	2: "CERT_TYPE_DEVICE_RESTRICTED_SIGNING",
	3: "CERT_TYPE_DEVICE_ENDORSEMENT_RSA",
	4: "CERT_TYPE_DEVICE_ECDH_EXCHANGE",
}

var ZEveCertType_value = map[string]int32{
	"CERT_TYPE_DEVICE_NONE":               0,
	"CERT_TYPE_DEVICE_ONBOARDING":         1,
	"CERT_TYPE_DEVICE_RESTRICTED_SIGNING": 2,
	"CERT_TYPE_DEVICE_ENDORSEMENT_RSA":    3,
	"CERT_TYPE_DEVICE_ECDH_EXCHANGE":      4,
}

func (x ZEveCertType) String() string {
	return proto.EnumName(ZEveCertType_name, int32(x))
}

func (ZEveCertType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{4}
}

// This is the request payload for POST /api/v2/edgeDevice/id/<uuid>/attest
// The message is assumed to be protected by signing envelope
type ZAttestReq struct {
	ReqType              ZAttestReqType `protobuf:"varint,1,opt,name=reqType,proto3,enum=ZAttestReqType" json:"reqType,omitempty"`
	Quote                *ZAttestQuote  `protobuf:"bytes,2,opt,name=quote,proto3" json:"quote,omitempty"`
	Certs                []*ZEveCert    `protobuf:"bytes,3,rep,name=certs,proto3" json:"certs,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *ZAttestReq) Reset()         { *m = ZAttestReq{} }
func (m *ZAttestReq) String() string { return proto.CompactTextString(m) }
func (*ZAttestReq) ProtoMessage()    {}
func (*ZAttestReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{0}
}

func (m *ZAttestReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZAttestReq.Unmarshal(m, b)
}
func (m *ZAttestReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZAttestReq.Marshal(b, m, deterministic)
}
func (m *ZAttestReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZAttestReq.Merge(m, src)
}
func (m *ZAttestReq) XXX_Size() int {
	return xxx_messageInfo_ZAttestReq.Size(m)
}
func (m *ZAttestReq) XXX_DiscardUnknown() {
	xxx_messageInfo_ZAttestReq.DiscardUnknown(m)
}

var xxx_messageInfo_ZAttestReq proto.InternalMessageInfo

func (m *ZAttestReq) GetReqType() ZAttestReqType {
	if m != nil {
		return m.ReqType
	}
	return ZAttestReqType_ATTEST_REQ_NONE
}

func (m *ZAttestReq) GetQuote() *ZAttestQuote {
	if m != nil {
		return m.Quote
	}
	return nil
}

func (m *ZAttestReq) GetCerts() []*ZEveCert {
	if m != nil {
		return m.Certs
	}
	return nil
}

// This is the response payload for POST /api/v2/edgeDevice/id/<uuid>/attest
// The message is assumed to be protected by signing envelope
type ZAttestResponse struct {
	RespType             ZAttestRespType   `protobuf:"varint,1,opt,name=respType,proto3,enum=ZAttestRespType" json:"respType,omitempty"`
	Nonce                *ZAttestNonceResp `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	QuoteResp            *ZAttestQuoteResp `protobuf:"bytes,3,opt,name=quoteResp,proto3" json:"quoteResp,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ZAttestResponse) Reset()         { *m = ZAttestResponse{} }
func (m *ZAttestResponse) String() string { return proto.CompactTextString(m) }
func (*ZAttestResponse) ProtoMessage()    {}
func (*ZAttestResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{1}
}

func (m *ZAttestResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZAttestResponse.Unmarshal(m, b)
}
func (m *ZAttestResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZAttestResponse.Marshal(b, m, deterministic)
}
func (m *ZAttestResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZAttestResponse.Merge(m, src)
}
func (m *ZAttestResponse) XXX_Size() int {
	return xxx_messageInfo_ZAttestResponse.Size(m)
}
func (m *ZAttestResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ZAttestResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ZAttestResponse proto.InternalMessageInfo

func (m *ZAttestResponse) GetRespType() ZAttestRespType {
	if m != nil {
		return m.RespType
	}
	return ZAttestRespType_ATTEST_RESP_NONE
}

func (m *ZAttestResponse) GetNonce() *ZAttestNonceResp {
	if m != nil {
		return m.Nonce
	}
	return nil
}

func (m *ZAttestResponse) GetQuoteResp() *ZAttestQuoteResp {
	if m != nil {
		return m.QuoteResp
	}
	return nil
}

type ZAttestNonceResp struct {
	Nonce                []byte   `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZAttestNonceResp) Reset()         { *m = ZAttestNonceResp{} }
func (m *ZAttestNonceResp) String() string { return proto.CompactTextString(m) }
func (*ZAttestNonceResp) ProtoMessage()    {}
func (*ZAttestNonceResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{2}
}

func (m *ZAttestNonceResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZAttestNonceResp.Unmarshal(m, b)
}
func (m *ZAttestNonceResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZAttestNonceResp.Marshal(b, m, deterministic)
}
func (m *ZAttestNonceResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZAttestNonceResp.Merge(m, src)
}
func (m *ZAttestNonceResp) XXX_Size() int {
	return xxx_messageInfo_ZAttestNonceResp.Size(m)
}
func (m *ZAttestNonceResp) XXX_DiscardUnknown() {
	xxx_messageInfo_ZAttestNonceResp.DiscardUnknown(m)
}

var xxx_messageInfo_ZAttestNonceResp proto.InternalMessageInfo

func (m *ZAttestNonceResp) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

type ZAttestQuote struct {
	AttestData []byte `protobuf:"bytes,1,opt,name=attestData,proto3" json:"attestData,omitempty"`
	//nonce is included in attestData
	Signature            []byte   `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZAttestQuote) Reset()         { *m = ZAttestQuote{} }
func (m *ZAttestQuote) String() string { return proto.CompactTextString(m) }
func (*ZAttestQuote) ProtoMessage()    {}
func (*ZAttestQuote) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{3}
}

func (m *ZAttestQuote) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZAttestQuote.Unmarshal(m, b)
}
func (m *ZAttestQuote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZAttestQuote.Marshal(b, m, deterministic)
}
func (m *ZAttestQuote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZAttestQuote.Merge(m, src)
}
func (m *ZAttestQuote) XXX_Size() int {
	return xxx_messageInfo_ZAttestQuote.Size(m)
}
func (m *ZAttestQuote) XXX_DiscardUnknown() {
	xxx_messageInfo_ZAttestQuote.DiscardUnknown(m)
}

var xxx_messageInfo_ZAttestQuote proto.InternalMessageInfo

func (m *ZAttestQuote) GetAttestData() []byte {
	if m != nil {
		return m.AttestData
	}
	return nil
}

func (m *ZAttestQuote) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type ZAttestQuoteResp struct {
	Response             ZAttestResponseCode `protobuf:"varint,1,opt,name=response,proto3,enum=ZAttestResponseCode" json:"response,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *ZAttestQuoteResp) Reset()         { *m = ZAttestQuoteResp{} }
func (m *ZAttestQuoteResp) String() string { return proto.CompactTextString(m) }
func (*ZAttestQuoteResp) ProtoMessage()    {}
func (*ZAttestQuoteResp) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{4}
}

func (m *ZAttestQuoteResp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZAttestQuoteResp.Unmarshal(m, b)
}
func (m *ZAttestQuoteResp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZAttestQuoteResp.Marshal(b, m, deterministic)
}
func (m *ZAttestQuoteResp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZAttestQuoteResp.Merge(m, src)
}
func (m *ZAttestQuoteResp) XXX_Size() int {
	return xxx_messageInfo_ZAttestQuoteResp.Size(m)
}
func (m *ZAttestQuoteResp) XXX_DiscardUnknown() {
	xxx_messageInfo_ZAttestQuoteResp.DiscardUnknown(m)
}

var xxx_messageInfo_ZAttestQuoteResp proto.InternalMessageInfo

func (m *ZAttestQuoteResp) GetResponse() ZAttestResponseCode {
	if m != nil {
		return m.Response
	}
	return ZAttestResponseCode_ATTEST_RESPONSE_NONE
}

type ZEveCert struct {
	HashAlgo             ZEveCertHashType `protobuf:"varint,1,opt,name=hashAlgo,proto3,enum=ZEveCertHashType" json:"hashAlgo,omitempty"`
	CertHash             []byte           `protobuf:"bytes,2,opt,name=certHash,proto3" json:"certHash,omitempty"`
	Type                 ZEveCertType     `protobuf:"varint,3,opt,name=type,proto3,enum=ZEveCertType" json:"type,omitempty"`
	Cert                 []byte           `protobuf:"bytes,4,opt,name=cert,proto3" json:"cert,omitempty"`
	Attributes           *ZEveCertAttr    `protobuf:"bytes,5,opt,name=attributes,proto3" json:"attributes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *ZEveCert) Reset()         { *m = ZEveCert{} }
func (m *ZEveCert) String() string { return proto.CompactTextString(m) }
func (*ZEveCert) ProtoMessage()    {}
func (*ZEveCert) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{5}
}

func (m *ZEveCert) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZEveCert.Unmarshal(m, b)
}
func (m *ZEveCert) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZEveCert.Marshal(b, m, deterministic)
}
func (m *ZEveCert) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZEveCert.Merge(m, src)
}
func (m *ZEveCert) XXX_Size() int {
	return xxx_messageInfo_ZEveCert.Size(m)
}
func (m *ZEveCert) XXX_DiscardUnknown() {
	xxx_messageInfo_ZEveCert.DiscardUnknown(m)
}

var xxx_messageInfo_ZEveCert proto.InternalMessageInfo

func (m *ZEveCert) GetHashAlgo() ZEveCertHashType {
	if m != nil {
		return m.HashAlgo
	}
	return ZEveCertHashType_HASH_NONE
}

func (m *ZEveCert) GetCertHash() []byte {
	if m != nil {
		return m.CertHash
	}
	return nil
}

func (m *ZEveCert) GetType() ZEveCertType {
	if m != nil {
		return m.Type
	}
	return ZEveCertType_CERT_TYPE_DEVICE_NONE
}

func (m *ZEveCert) GetCert() []byte {
	if m != nil {
		return m.Cert
	}
	return nil
}

func (m *ZEveCert) GetAttributes() *ZEveCertAttr {
	if m != nil {
		return m.Attributes
	}
	return nil
}

type ZEveCertAttr struct {
	IsMutable            bool     `protobuf:"varint,1,opt,name=isMutable,proto3" json:"isMutable,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZEveCertAttr) Reset()         { *m = ZEveCertAttr{} }
func (m *ZEveCertAttr) String() string { return proto.CompactTextString(m) }
func (*ZEveCertAttr) ProtoMessage()    {}
func (*ZEveCertAttr) Descriptor() ([]byte, []int) {
	return fileDescriptor_208cf26448842a3f, []int{6}
}

func (m *ZEveCertAttr) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZEveCertAttr.Unmarshal(m, b)
}
func (m *ZEveCertAttr) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZEveCertAttr.Marshal(b, m, deterministic)
}
func (m *ZEveCertAttr) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZEveCertAttr.Merge(m, src)
}
func (m *ZEveCertAttr) XXX_Size() int {
	return xxx_messageInfo_ZEveCertAttr.Size(m)
}
func (m *ZEveCertAttr) XXX_DiscardUnknown() {
	xxx_messageInfo_ZEveCertAttr.DiscardUnknown(m)
}

var xxx_messageInfo_ZEveCertAttr proto.InternalMessageInfo

func (m *ZEveCertAttr) GetIsMutable() bool {
	if m != nil {
		return m.IsMutable
	}
	return false
}

func init() {
	proto.RegisterEnum("ZAttestReqType", ZAttestReqType_name, ZAttestReqType_value)
	proto.RegisterEnum("ZAttestRespType", ZAttestRespType_name, ZAttestRespType_value)
	proto.RegisterEnum("ZAttestResponseCode", ZAttestResponseCode_name, ZAttestResponseCode_value)
	proto.RegisterEnum("ZEveCertHashType", ZEveCertHashType_name, ZEveCertHashType_value)
	proto.RegisterEnum("ZEveCertType", ZEveCertType_name, ZEveCertType_value)
	proto.RegisterType((*ZAttestReq)(nil), "ZAttestReq")
	proto.RegisterType((*ZAttestResponse)(nil), "ZAttestResponse")
	proto.RegisterType((*ZAttestNonceResp)(nil), "ZAttestNonceResp")
	proto.RegisterType((*ZAttestQuote)(nil), "ZAttestQuote")
	proto.RegisterType((*ZAttestQuoteResp)(nil), "ZAttestQuoteResp")
	proto.RegisterType((*ZEveCert)(nil), "ZEveCert")
	proto.RegisterType((*ZEveCertAttr)(nil), "ZEveCertAttr")
}

func init() { proto.RegisterFile("attest.proto", fileDescriptor_208cf26448842a3f) }

var fileDescriptor_208cf26448842a3f = []byte{
	// 689 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x94, 0x5d, 0x4f, 0xea, 0x48,
	0x1c, 0xc6, 0xb7, 0xbc, 0xec, 0xc2, 0x5f, 0xd4, 0x71, 0xc4, 0x95, 0xd5, 0x8d, 0xb2, 0xd5, 0x44,
	0x96, 0x68, 0xd9, 0x65, 0xb3, 0x5e, 0xec, 0x5d, 0x2d, 0xb3, 0x40, 0xa2, 0x45, 0xa6, 0xf5, 0xe4,
	0x1c, 0x6f, 0x9a, 0x02, 0x23, 0x90, 0x20, 0xc5, 0x76, 0x30, 0xf1, 0x24, 0xe7, 0x83, 0x9c, 0xcf,
	0x72, 0x72, 0xbe, 0xdb, 0x49, 0xa7, 0x2f, 0x94, 0xe2, 0x5d, 0xe7, 0x79, 0x7e, 0xff, 0x17, 0x9f,
	0x19, 0x81, 0x92, 0xcd, 0x39, 0xf3, 0xb8, 0xb2, 0x70, 0x1d, 0xee, 0xc8, 0x5f, 0x00, 0x1e, 0x55,
	0x21, 0x50, 0xf6, 0x82, 0xff, 0x84, 0x5f, 0x5c, 0xf6, 0x62, 0xbe, 0x2d, 0x58, 0x45, 0xaa, 0x4a,
	0xb5, 0x9d, 0xe6, 0xae, 0xb2, 0x72, 0x7d, 0x99, 0x46, 0x3e, 0x3e, 0x83, 0xfc, 0xcb, 0xd2, 0xe1,
	0xac, 0x92, 0xa9, 0x4a, 0xb5, 0xad, 0xe6, 0x76, 0x04, 0xf6, 0x7d, 0x91, 0x06, 0x1e, 0x3e, 0x85,
	0xfc, 0x90, 0xb9, 0xdc, 0xab, 0x64, 0xab, 0xd9, 0xda, 0x56, 0xb3, 0xa8, 0x3c, 0x92, 0x57, 0xa6,
	0x31, 0x97, 0xd3, 0x40, 0x97, 0xbf, 0x4a, 0xb0, 0x1b, 0x4f, 0xf0, 0x16, 0xce, 0xdc, 0x63, 0xf8,
	0x12, 0x0a, 0x2e, 0xf3, 0x16, 0x89, 0x2d, 0x90, 0x92, 0x60, 0xc4, 0x1a, 0x31, 0x81, 0x2f, 0x20,
	0x3f, 0x77, 0xe6, 0xc3, 0x68, 0x8f, 0xbd, 0x08, 0xd5, 0x7d, 0xd1, 0xe7, 0x69, 0xe0, 0xe3, 0x06,
	0x14, 0xc5, 0x52, 0xbe, 0x56, 0xc9, 0xae, 0xc3, 0xfd, 0xc8, 0xa0, 0x2b, 0x46, 0xae, 0x01, 0x4a,
	0xf7, 0xc2, 0xe5, 0x68, 0x9a, 0xbf, 0x58, 0x29, 0x6c, 0x2d, 0xdf, 0x42, 0x29, 0xd9, 0x08, 0x9f,
	0x00, 0x04, 0x21, 0xb7, 0x6c, 0x6e, 0x87, 0x68, 0x42, 0xc1, 0xbf, 0x43, 0xd1, 0x9b, 0x8e, 0xe7,
	0x36, 0x5f, 0xba, 0xc1, 0xde, 0x25, 0xba, 0x12, 0xe4, 0x56, 0x3c, 0x37, 0x5e, 0x0b, 0xff, 0x15,
	0x64, 0xe2, 0xe7, 0x13, 0x66, 0x52, 0x56, 0x52, 0xb9, 0x69, 0xce, 0x28, 0xcc, 0xc5, 0x3f, 0xc9,
	0xdf, 0x24, 0x28, 0x44, 0x69, 0xe3, 0x2b, 0x28, 0x4c, 0x6c, 0x6f, 0xa2, 0xce, 0xc6, 0x4e, 0x58,
	0xbe, 0x17, 0x5f, 0x45, 0xc7, 0xf6, 0x26, 0x41, 0xa6, 0x11, 0x82, 0x8f, 0xa0, 0x30, 0x0c, 0x9d,
	0x70, 0xbd, 0xf8, 0x8c, 0xff, 0x80, 0x1c, 0xf7, 0x6f, 0x26, 0x2b, 0xda, 0x6c, 0xc7, 0x6d, 0x44,
	0x0b, 0x61, 0x61, 0x0c, 0x39, 0x1f, 0xaf, 0xe4, 0x44, 0xa9, 0xf8, 0xc6, 0x57, 0x22, 0x12, 0x77,
	0x3a, 0x58, 0x72, 0xe6, 0x55, 0xf2, 0xd1, 0x9b, 0x09, 0x8b, 0x55, 0xce, 0x5d, 0x9a, 0x00, 0xe4,
	0x4b, 0x28, 0x25, 0x3d, 0x3f, 0xb1, 0xa9, 0x77, 0xb7, 0xe4, 0xf6, 0x60, 0x16, 0x04, 0x50, 0xa0,
	0x2b, 0xa1, 0xfe, 0x04, 0x3b, 0xeb, 0xcf, 0x14, 0xef, 0xc3, 0xae, 0x6a, 0x9a, 0xc4, 0x30, 0x2d,
	0x4a, 0xfa, 0x96, 0xde, 0xd3, 0x09, 0xfa, 0x29, 0x25, 0x6a, 0x84, 0x9a, 0x48, 0xc2, 0x65, 0x40,
	0xeb, 0xa4, 0x46, 0x50, 0x26, 0xa5, 0xf6, 0x1f, 0x7a, 0x26, 0x41, 0xd9, 0xfa, 0x62, 0xed, 0xb1,
	0x8a, 0x41, 0x49, 0xd0, 0xb8, 0x8f, 0x26, 0xa5, 0xd4, 0x70, 0xd4, 0x01, 0xec, 0xa5, 0x58, 0x31,
	0xeb, 0x08, 0x7e, 0x4d, 0xca, 0x62, 0x98, 0xf8, 0x44, 0xd9, 0xfa, 0x14, 0xf6, 0xdf, 0xb9, 0x66,
	0x5c, 0x81, 0x72, 0xa2, 0xa4, 0xa7, 0x1b, 0x24, 0x9a, 0x7c, 0x0c, 0x87, 0x69, 0xc7, 0x78, 0xd0,
	0x34, 0x62, 0x18, 0x48, 0x7a, 0xcf, 0xfc, 0x5f, 0xed, 0xde, 0x3e, 0x50, 0x82, 0x32, 0xf5, 0xff,
	0x00, 0xa5, 0x9f, 0x04, 0xde, 0x86, 0x62, 0x47, 0x35, 0x3a, 0x51, 0xf3, 0x43, 0xd8, 0x17, 0x47,
	0xa3, 0xa3, 0x36, 0xff, 0xbd, 0xb6, 0xfe, 0xbe, 0x1e, 0xbc, 0x71, 0xe6, 0x21, 0xa9, 0xfe, 0x5d,
	0x5a, 0xdd, 0x97, 0x28, 0xfc, 0x0d, 0x0e, 0xfc, 0x3f, 0xda, 0x32, 0x3f, 0xdd, 0x13, 0xab, 0x45,
	0x3e, 0x74, 0xb5, 0x78, 0xc3, 0x53, 0x38, 0xde, 0xb0, 0x7a, 0xfa, 0x4d, 0x4f, 0xa5, 0xad, 0xae,
	0xde, 0x46, 0x12, 0xbe, 0x80, 0xb3, 0x0d, 0x80, 0x12, 0xc3, 0xa4, 0x5d, 0xcd, 0x24, 0x2d, 0xcb,
	0xe8, 0xb6, 0x75, 0x1f, 0xcc, 0xe0, 0x73, 0xa8, 0x6e, 0x80, 0x44, 0x6f, 0xf5, 0xa8, 0x41, 0xee,
	0x88, 0x6e, 0x5a, 0xd4, 0x50, 0x51, 0x16, 0xcb, 0x70, 0xb2, 0x49, 0x69, 0xad, 0x8e, 0x45, 0x3e,
	0x6a, 0x1d, 0x55, 0x6f, 0x13, 0x94, 0xbb, 0x69, 0xc3, 0xe9, 0xd0, 0x79, 0x56, 0x3e, 0xb3, 0x11,
	0x1b, 0xd9, 0xca, 0x70, 0xe6, 0x2c, 0x47, 0xca, 0xd2, 0x63, 0xee, 0xeb, 0x74, 0xc8, 0x82, 0x1f,
	0xca, 0xc7, 0xf3, 0xf1, 0x94, 0x4f, 0x96, 0x03, 0x65, 0xe8, 0x3c, 0x37, 0x66, 0x4f, 0x57, 0x6c,
	0x34, 0x66, 0x0d, 0xf6, 0xca, 0x1a, 0xf6, 0x62, 0xda, 0x18, 0x3b, 0x8d, 0xe0, 0xbf, 0x7b, 0xf0,
	0xb3, 0x80, 0xff, 0xf9, 0x11, 0x00, 0x00, 0xff, 0xff, 0xe9, 0xcb, 0x48, 0x86, 0x65, 0x05, 0x00,
	0x00,
}