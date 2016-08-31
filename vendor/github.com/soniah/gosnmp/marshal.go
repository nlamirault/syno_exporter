// Copyright 2012-2014 The GoSNMP Authors. All rights reserved.  Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.

package gosnmp

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"encoding/asn1"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

//
// Remaining globals and definitions located here.
// See http://www.rane.com/note161.html for a succint description of the SNMP
// protocol.
//

// SnmpVersion 1, 2c and 3 implemented
type SnmpVersion uint8

// SnmpVersion 1, 2c and 3 implemented
const (
	Version1  SnmpVersion = 0x0
	Version2c SnmpVersion = 0x1
	Version3  SnmpVersion = 0x3
)

// SnmpV3MsgFlags contains various message flags to describe Authentication, Privacy, and whether a report PDU must be sent.
type SnmpV3MsgFlags uint8

// Possible values of SnmpV3MsgFlags
const (
	NoAuthNoPriv SnmpV3MsgFlags = 0x0 // No authentication, and no privacy
	AuthNoPriv   SnmpV3MsgFlags = 0x1 // Authentication and no privacy
	AuthPriv     SnmpV3MsgFlags = 0x3 // Authentication and privacy
	Reportable   SnmpV3MsgFlags = 0x4 // Report PDU must be sent.
)

// SnmpV3SecurityModel describes the security model used by a SnmpV3 connection
type SnmpV3SecurityModel uint8

// UserSecurityModel is the only SnmpV3SecurityModel currently implemented.
const (
	UserSecurityModel SnmpV3SecurityModel = 3
)

// SnmpV3AuthProtocol describes the authentication protocol in use by an authenticated SnmpV3 connection.
type SnmpV3AuthProtocol uint8

// NoAuth, MD5, and SHA are implemented
const (
	NoAuth SnmpV3AuthProtocol = 1
	MD5    SnmpV3AuthProtocol = 2
	SHA    SnmpV3AuthProtocol = 3
)

// SnmpV3PrivProtocol is the privacy protocol in use by an private SnmpV3 connection.
type SnmpV3PrivProtocol uint8

// NoPriv, DES implemented, AES planned
const (
	NoPriv SnmpV3PrivProtocol = 1
	DES    SnmpV3PrivProtocol = 2
	AES    SnmpV3PrivProtocol = 3
)

// SnmpV3SecurityParameters is a generic interface type to contain various implementations of SnmpV3SecurityParameters
type SnmpV3SecurityParameters interface {
	Copy() SnmpV3SecurityParameters
}

// UsmSecurityParameters is an implementation of SnmpV3SecurityParameters for the UserSecurityModel
type UsmSecurityParameters struct {
	AuthoritativeEngineID    string
	AuthoritativeEngineBoots uint32
	AuthoritativeEngineTime  uint32
	UserName                 string
	AuthenticationParameters string
	PrivacyParameters        []byte

	AuthenticationProtocol SnmpV3AuthProtocol
	PrivacyProtocol        SnmpV3PrivProtocol

	AuthenticationPassphrase string
	PrivacyPassphrase        string

	localDESSalt uint32
	localAESSalt uint64
}

// SnmpPacket struct represents the entire SNMP Message or Sequence at the
// application layer.
type SnmpPacket struct {
	Version            SnmpVersion
	MsgFlags           SnmpV3MsgFlags
	SecurityModel      SnmpV3SecurityModel
	SecurityParameters SnmpV3SecurityParameters
	ContextEngineID    string
	ContextName        string
	Community          string
	PDUType            PDUType
	MsgID              uint32
	RequestID          uint32
	Error              SNMPError
	ErrorIndex         uint8
	NonRepeaters       uint8
	MaxRepetitions     uint8
	Variables          []SnmpPDU
	Logger             Logger
}

// VarBind struct represents an SNMP Varbind.
type VarBind struct {
	Name  asn1.ObjectIdentifier
	Value asn1.RawValue
}

// PDUType describes which SNMP Protocol Data Unit is being sent.
type PDUType byte

// The currently supported PDUType's
const (
	Sequence       PDUType = 0x30
	GetRequest     PDUType = 0xa0
	GetNextRequest PDUType = 0xa1
	GetResponse    PDUType = 0xa2
	SetRequest     PDUType = 0xa3
	Trap           PDUType = 0xa4 // v1
	GetBulkRequest PDUType = 0xa5
	InformRequest  PDUType = 0xa6
	SNMPv2Trap     PDUType = 0xa7 // v2c, v3
	Report         PDUType = 0xa8
)

const rxBufSize = 65535 // max size of IPv4 & IPv6 packet

// Logger is an interface used for debugging. Both Print and
// Printf have the same interfaces as Package Log in the std library. The
// Logger interface is small to give you flexibility in how you do
// your debugging.
//
// For verbose logging to stdout:
//
//     gosnmp_logger = log.New(os.Stdout, "", 0)
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// send/receive one snmp request
func (x *GoSNMP) sendOneRequest(pdus []SnmpPDU, packetOut *SnmpPacket,
	wait bool) (result *SnmpPacket, err error) {
	finalDeadline := time.Now().Add(x.Timeout)

	allReqIDs := make([]uint32, 0, x.Retries+1)
	allMsgIDs := make([]uint32, 0, x.Retries+1)
	for retries := 0; ; retries++ {
		if retries > 0 {
			if x.loggingEnabled {
				x.Logger.Printf("Retry number %d. Last error was: %v", retries, err)
			}
			if time.Now().After(finalDeadline) {
				err = fmt.Errorf("Request timeout (after %d retries)", retries-1)
				break
			}
			if retries > x.Retries {
				// Report last error
				break
			}
		}
		err = nil

		reqDeadline := time.Now().Add(x.Timeout / time.Duration(x.Retries+1))
		x.Conn.SetDeadline(reqDeadline)

		// Request ID is an atomic counter (started at a random value)
		reqID := atomic.AddUint32(&(x.requestID), 1) // TODO: fix overflows
		allReqIDs = append(allReqIDs, reqID)

		var msgID uint32
		if x.Version == Version3 {
			packetOut, err = x.buildPacket3(msgID, allMsgIDs, packetOut)
			if err != nil {
				break
			}
		}

		var outBuf []byte
		outBuf, err = packetOut.marshalMsg(pdus, packetOut.PDUType, msgID, reqID)
		if err != nil {
			// Don't retry - not going to get any better!
			err = fmt.Errorf("marshal: %v", err)
			break
		}

		_, err = x.Conn.Write(outBuf)
		if err != nil {
			continue
		}

		// all sends wait for the return packet, except for SNMPv2Trap
		if wait == false {
			return &SnmpPacket{}, nil
		}

		for {
			// Receive response and try receiving again on any decoding error.
			// Let the deadline abort us if we don't receive a valid response.

			var resp []byte
			resp, err = x.receive()
			if err != nil {
				// receive error. retrying won't help. abort
				break
			}
			result = new(SnmpPacket)
			result.Logger = x.Logger
			result.MsgFlags = packetOut.MsgFlags
			if packetOut.SecurityParameters != nil {
				result.SecurityParameters = packetOut.SecurityParameters.Copy()
			}
			err = x.unmarshal(resp, result)
			if err != nil {
				err = fmt.Errorf("Unable to decode packet: %s", err.Error())
				continue
			}
			if result == nil || len(result.Variables) < 1 {
				err = fmt.Errorf("Unable to decode packet: nil")
				continue
			}

			validID := false
			for _, id := range allReqIDs {
				if id == result.RequestID {
					validID = true
				}
			}
			if result.RequestID == 0 {
				validID = true
			}
			if !validID {
				err = fmt.Errorf("Out of order response")
				continue
			}

			break
		}
		if err != nil {
			continue
		}

		// Success!
		return result, nil
	}

	// Return last error
	return nil, err
}

// generic "sender" that negotiate any version of snmp request
//
// all sends wait for the return packet, except for SNMPv2Trap
func (x *GoSNMP) send(pdus []SnmpPDU,
	packetOut *SnmpPacket, wait bool) (result *SnmpPacket, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("recover: %v", e)
		}
	}()

	if x.Conn == nil {
		return nil, fmt.Errorf("&GoSNMP.Conn is missing. Provide a connection or use Connect()")
	}

	if x.Retries < 0 {
		x.Retries = 0
	}

	if packetOut.Version == Version3 {
		if packetOut, err = x.setAuthoritativeEngine(packetOut, wait); err != nil {
			return &SnmpPacket{}, err
		}
	}

	// perform request
	result, err = x.sendOneRequest(pdus, packetOut, wait)
	if err != nil {
		return result, err
	}

	if result.Version == Version3 && result.SecurityModel == UserSecurityModel {
		result, err = x.setAuthoritativeEngine2(packetOut, result, pdus, wait)
	}
	return result, err
}

// -- Marshalling Logic --------------------------------------------------------

// marshal an SNMP message
func (packet *SnmpPacket) marshalMsg(pdus []SnmpPDU,
	pdutype PDUType, msgid uint32, requestid uint32) ([]byte, error) {
	var authParamStart uint32
	buf := new(bytes.Buffer)

	// version
	buf.Write([]byte{2, 1, byte(packet.Version)})

	if packet.Version != Version3 {
		// community
		buf.Write([]byte{4, uint8(len(packet.Community))})
		buf.WriteString(packet.Community)
		// pdu
		pdu, err := packet.marshalPDU(pdus, requestid)
		if err != nil {
			return nil, err
		}
		buf.Write(pdu)
	} else {
		header, err := packet.marshalSnmpV3Header(msgid)
		if err != nil {
			return nil, err
		}
		buf.Write([]byte{byte(Sequence), byte(len(header))})
		buf.Write(header)

		var securityParameters []byte
		if packet.SecurityModel == UserSecurityModel {
			securityParameters, authParamStart, err = packet.marshalSnmpV3UsmSecurityParameters()
			if err != nil {
				return nil, err
			}
		}

		buf.Write([]byte{byte(OctetString)})
		secParamLen, err := marshalLength(len(securityParameters))
		if err != nil {
			return nil, err
		}
		buf.Write(secParamLen)
		authParamStart += uint32(buf.Len())
		buf.Write(securityParameters)

		scopedPdu, err := packet.marshalSnmpV3ScopedPDU(pdus, requestid)
		if err != nil {
			return nil, err
		}
		buf.Write(scopedPdu)
	}

	// build up resulting msg - sequence, length then the tail (buf)
	msg := new(bytes.Buffer)
	msg.WriteByte(byte(Sequence))

	bufLengthBytes, err2 := marshalLength(buf.Len())
	if err2 != nil {
		return nil, err2
	}
	msg.Write(bufLengthBytes)
	authParamStart += uint32(msg.Len())
	buf.WriteTo(msg) // reverse logic - want to do msg.Write(buf)

	authenticatedMessage, err := packet.authenticate(msg.Bytes(), authParamStart)
	if err != nil {
		return nil, err
	}

	return authenticatedMessage, nil
}

// marshal a snmp version 3 packet header
func (packet *SnmpPacket) marshalSnmpV3Header(msgid uint32) ([]byte, error) {
	buf := new(bytes.Buffer)

	// msg id
	buf.Write([]byte{byte(Integer), 4})
	err := binary.Write(buf, binary.BigEndian, msgid)
	if err != nil {
		return nil, err
	}

	// maximum response msg size
	maxmsgsize := marshalUvarInt(rxBufSize)
	buf.Write([]byte{byte(Integer), byte(len(maxmsgsize))})
	buf.Write(maxmsgsize)

	// msg flags
	buf.Write([]byte{byte(OctetString), 1, byte(packet.MsgFlags)})

	// msg security model
	buf.Write([]byte{byte(Integer), 1, byte(packet.SecurityModel)})

	return buf.Bytes(), nil
}

// marshal a snmp version 3 security parameters field for the User Security Model
func (packet *SnmpPacket) marshalSnmpV3UsmSecurityParameters() ([]byte, uint32, error) {
	var buf bytes.Buffer
	var authParamStart uint32

	secParams, ok := packet.SecurityParameters.(*UsmSecurityParameters)
	if !ok || secParams == nil {
		return nil, 0, fmt.Errorf("packet.SecurityParameters is not of type &UsmSecurityParameters")
	}

	// msgAuthoritativeEngineID
	buf.Write([]byte{byte(OctetString), byte(len(secParams.AuthoritativeEngineID))})
	buf.WriteString(secParams.AuthoritativeEngineID)

	// msgAuthoritativeEngineBoots
	msgAuthoritativeEngineBoots := marshalUvarInt(secParams.AuthoritativeEngineBoots)
	buf.Write([]byte{byte(Integer), byte(len(msgAuthoritativeEngineBoots))})
	buf.Write(msgAuthoritativeEngineBoots)

	// msgAuthoritativeEngineTime
	msgAuthoritativeEngineTime := marshalUvarInt(secParams.AuthoritativeEngineTime)
	buf.Write([]byte{byte(Integer), byte(len(msgAuthoritativeEngineTime))})
	buf.Write(msgAuthoritativeEngineTime)

	// msgUserName
	buf.Write([]byte{byte(OctetString), byte(len(secParams.UserName))})
	buf.WriteString(secParams.UserName)

	authParamStart = uint32(buf.Len() + 2) // +2 indicates PDUType + Length
	// msgAuthenticationParameters
	if packet.MsgFlags&AuthNoPriv > 0 {
		buf.Write([]byte{byte(OctetString), 12,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0, 0, 0})
	} else {
		buf.Write([]byte{byte(OctetString), 0})
	}
	// msgPrivacyParameters
	if packet.MsgFlags&AuthPriv > AuthNoPriv {
		privlen, err := marshalLength(len(secParams.PrivacyParameters))
		if err != nil {
			return nil, 0, err
		}
		buf.Write([]byte{byte(OctetString)})
		buf.Write(privlen)
		buf.Write(secParams.PrivacyParameters)
	} else {
		buf.Write([]byte{byte(OctetString), 0})
	}

	// wrap security parameters in a sequence
	paramLen, err := marshalLength(buf.Len())
	if err != nil {
		return nil, 0, err
	}
	tmpseq := append([]byte{byte(Sequence)}, paramLen...)
	authParamStart += uint32(len(tmpseq))
	tmpseq = append(tmpseq, buf.Bytes()...)

	return tmpseq, authParamStart, nil
}

// marshal and encrypt (if necessary) a snmp version 3 Scoped PDU
func (packet *SnmpPacket) marshalSnmpV3ScopedPDU(pdus []SnmpPDU, requestid uint32) ([]byte, error) {
	var b []byte

	scopedPdu, err := packet.prepareSnmpV3ScopedPDU(pdus, requestid)
	if err != nil {
		return nil, err
	}
	pduLen, err := marshalLength(len(scopedPdu))
	if err != nil {
		return nil, err
	}
	b = append([]byte{byte(Sequence)}, pduLen...)
	scopedPdu = append(b, scopedPdu...)
	if packet.MsgFlags&AuthPriv > AuthNoPriv && packet.SecurityModel == UserSecurityModel {
		secParams, ok := packet.SecurityParameters.(*UsmSecurityParameters)
		if !ok || secParams == nil {
			return nil, fmt.Errorf("packet.SecurityModel indicates the User Security Model, but packet.SecurityParameters is not of type &UsmSecurityParameters")
		}
		var privkey = genlocalkey(secParams.AuthenticationProtocol,
			secParams.PrivacyPassphrase,
			secParams.AuthoritativeEngineID)
		switch secParams.PrivacyProtocol {
		case AES:
			var iv [16]byte
			binary.BigEndian.PutUint32(iv[:], secParams.AuthoritativeEngineBoots)
			binary.BigEndian.PutUint32(iv[4:], secParams.AuthoritativeEngineTime)
			copy(iv[8:], secParams.PrivacyParameters)

			block, err := aes.NewCipher(privkey[:16])
			if err != nil {
				return nil, err
			}
			stream := cipher.NewCFBEncrypter(block, iv[:])
			ciphertext := make([]byte, len(scopedPdu))
			stream.XORKeyStream(ciphertext, scopedPdu)
			pduLen, err := marshalLength(len(ciphertext))
			if err != nil {
				return nil, err
			}
			b = append([]byte{byte(OctetString)}, pduLen...)
			scopedPdu = append(b, ciphertext...)
		default:
			preiv := privkey[8:]
			var iv [8]byte
			for i := 0; i < len(iv); i++ {
				iv[i] = preiv[i] ^ secParams.PrivacyParameters[i]
			}
			block, err := des.NewCipher(privkey[:8])
			if err != nil {
				return nil, err
			}
			mode := cipher.NewCBCEncrypter(block, iv[:])

			pad := make([]byte, des.BlockSize-len(scopedPdu)%des.BlockSize)
			scopedPdu = append(scopedPdu, pad...)

			ciphertext := make([]byte, len(scopedPdu))
			mode.CryptBlocks(ciphertext, scopedPdu)
			pduLen, err := marshalLength(len(ciphertext))
			if err != nil {
				return nil, err
			}
			b = append([]byte{byte(OctetString)}, pduLen...)
			scopedPdu = append(b, ciphertext...)
		}

	}

	return scopedPdu, nil
}

// prepare the plain text of a snmp version 3 Scoped PDU
func (packet *SnmpPacket) prepareSnmpV3ScopedPDU(pdus []SnmpPDU, requestid uint32) ([]byte, error) {
	var buf bytes.Buffer

	//ContextEngineID
	idlen, err := marshalLength(len(packet.ContextEngineID))
	if err != nil {
		return nil, err
	}
	buf.Write(append([]byte{byte(OctetString)}, idlen...))
	buf.WriteString(packet.ContextEngineID)

	//ContextName
	namelen, err := marshalLength(len(packet.ContextName))
	if err != nil {
		return nil, err
	}
	buf.Write(append([]byte{byte(OctetString)}, namelen...))
	buf.WriteString(packet.ContextName)

	data, err := packet.marshalPDU(pdus, requestid)
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	return buf.Bytes(), nil
}

// marshal a PDU
func (packet *SnmpPacket) marshalPDU(pdus []SnmpPDU, requestid uint32) ([]byte, error) {
	buf := new(bytes.Buffer)

	// requestid
	buf.Write([]byte{2, 4})
	err := binary.Write(buf, binary.BigEndian, requestid)
	if err != nil {
		return nil, err
	}

	if packet.PDUType == GetBulkRequest {
		// non repeaters
		buf.Write([]byte{2, 1, packet.NonRepeaters})

		// max repetitions
		buf.Write([]byte{2, 1, packet.MaxRepetitions})
	} else { // get and getnext have same packet format

		// error
		buf.Write([]byte{2, 1, 0})

		// error index
		buf.Write([]byte{2, 1, 0})
	}

	// varbind list
	vbl, err := packet.marshalVBL(pdus)
	if err != nil {
		return nil, err
	}
	buf.Write(vbl)

	// build up resulting pdu - request type, length, then the tail (buf)
	pdu := new(bytes.Buffer)
	pdu.WriteByte(byte(packet.PDUType))

	bufLengthBytes, err2 := marshalLength(buf.Len())
	if err2 != nil {
		return nil, err2
	}
	pdu.Write(bufLengthBytes)

	buf.WriteTo(pdu) // reverse logic - want to do pdu.Write(buf)
	return pdu.Bytes(), nil
}

// marshal a varbind list
func (packet *SnmpPacket) marshalVBL(pdus []SnmpPDU) ([]byte, error) {

	vblBuf := new(bytes.Buffer)
	for _, pdu := range pdus {
		vb, err := marshalVarbind(&pdu)
		if err != nil {
			return nil, err
		}
		vblBuf.Write(vb)
	}

	vblBytes := vblBuf.Bytes()
	vblLengthBytes, err := marshalLength(len(vblBytes))
	if err != nil {
		return nil, err
	}

	// FIX does bytes.Buffer give better performance than byte slices?
	result := []byte{byte(Sequence)}
	result = append(result, vblLengthBytes...)
	result = append(result, vblBytes...)
	return result, nil
}

// marshal a varbind
func marshalVarbind(pdu *SnmpPDU) ([]byte, error) {
	oid, err := marshalOID(pdu.Name)
	if err != nil {
		return nil, err
	}
	pduBuf := new(bytes.Buffer)
	tmpBuf := new(bytes.Buffer)

	// Marshal the PDU type into the appropriate BER
	switch pdu.Type {

	case Null:
		pduBuf.Write([]byte{byte(Sequence), byte(len(oid) + 4)})
		pduBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		pduBuf.Write(oid)
		pduBuf.Write([]byte{Null, 0x00})

	/*
		NUMBERS:

		Integer32 and INTEGER:
		-2^31 and 2^31-1 inclusive (-2147483648 to 2147483647 decimal)

		Counter32, Gauge32, TimeTicks, Unsigned32:
		non-negative integer, maximum value of 2^32-1 (4294967295 decimal)
	*/

	case Integer:
		// TODO tests currently only cover positive integers

		// Oid
		tmpBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		tmpBuf.Write(oid)

		// Number
		var intBytes []byte
		switch value := pdu.Value.(type) {
		case byte:
			intBytes = []byte{byte(pdu.Value.(int))}
		case int:
			intBytes, err = marshalInt16(value)
			pdu.Check(err)
		default:
			return nil, fmt.Errorf("Unable to marshal PDU Integer; not byte or int.")
		}
		tmpBuf.Write([]byte{byte(Integer), byte(len(intBytes))})
		tmpBuf.Write(intBytes)

		// Sequence, length of oid + integer, then oid/integer data
		pduBuf.WriteByte(byte(Sequence))
		pduBuf.WriteByte(byte(len(oid) + len(intBytes) + 4))
		pduBuf.Write(tmpBuf.Bytes())

	case Counter32, Gauge32, TimeTicks, Uinteger32:
		// Oid
		tmpBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		tmpBuf.Write(oid)

		// Number
		var intBytes []byte
		switch value := pdu.Value.(type) {
		case uint32:
			intBytes, err = marshalUint32(value)
			pdu.Check(err)
		default:
			return nil, fmt.Errorf("Unable to marshal pdu.Type %v; unknown pdu.Value %v", pdu.Type, pdu.Value)
		}
		tmpBuf.Write([]byte{byte(pdu.Type), byte(len(intBytes))})
		tmpBuf.Write(intBytes)

		// Sequence, length of oid + integer, then oid/integer data
		pduBuf.WriteByte(byte(Sequence))
		pduBuf.WriteByte(byte(len(oid) + len(intBytes) + 4))
		pduBuf.Write(tmpBuf.Bytes())

	case OctetString:

		//Oid
		tmpBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		tmpBuf.Write(oid)

		//OctetString
		var octetStringBytes []byte
		switch value := pdu.Value.(type) {
		case []byte:
			octetStringBytes = value
		case string:
			octetStringBytes = []byte(value)
		default:
			return nil, fmt.Errorf("Unable to marshal PDU OctetString; not []byte or String.")
		}
		tmpBuf.Write([]byte{byte(OctetString), byte(len(octetStringBytes))})
		tmpBuf.Write(octetStringBytes)

		// Sequence, length of oid + octetstring, then oid/octetstring data
		pduBuf.WriteByte(byte(Sequence))
		pduBuf.WriteByte(byte(len(oid) + len(octetStringBytes) + 4))
		pduBuf.Write(tmpBuf.Bytes())

	case ObjectIdentifier:

		//Oid
		tmpBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		tmpBuf.Write(oid)
		value := pdu.Value.(string)
		oidBytes, err := marshalOID(value)
		pdu.Check(err)

		//Oid data
		tmpBuf.Write([]byte{byte(pdu.Type), byte(len(oidBytes))})
		tmpBuf.Write(oidBytes)

		// Sequence, length of oid + oid, then oid/oid data
		pduBuf.WriteByte(byte(Sequence))
		pduBuf.WriteByte(byte(len(oid) + len(oidBytes) + 4))
		pduBuf.Write(tmpBuf.Bytes())

	// MrSpock changes. TODO NO tests for this yet - waiting for .pcap
	case IPAddress:
		//Oid
		tmpBuf.Write([]byte{byte(ObjectIdentifier), byte(len(oid))})
		tmpBuf.Write(oid)
		//OctetString
		var ipAddressBytes []byte
		switch value := pdu.Value.(type) {
		case []byte:
			ipAddressBytes = value
		case string:
			ip := net.ParseIP(value)
			ipAddressBytes = ipv4toBytes(ip)
		default:
			return nil, fmt.Errorf("Unable to marshal PDU IPAddress; not []byte or String.")
		}
		tmpBuf.Write([]byte{byte(IPAddress), byte(len(ipAddressBytes))})
		tmpBuf.Write(ipAddressBytes)
		// Sequence, length of oid + octetstring, then oid/octetstring data
		pduBuf.WriteByte(byte(Sequence))
		pduBuf.WriteByte(byte(len(oid) + len(ipAddressBytes) + 4))
		pduBuf.Write(tmpBuf.Bytes())

	default:
		return nil, fmt.Errorf("Unable to marshal PDU: unknown BER type %q", pdu.Type)
	}

	return pduBuf.Bytes(), nil
}

// -- Unmarshalling Logic ------------------------------------------------------

func (x *GoSNMP) unmarshal(packet []byte, response *SnmpPacket) error {
	if response == nil {
		return fmt.Errorf("Cannot unmarshal response into nil packet reference")
	}
	var OrigMsgFlags = response.MsgFlags
	var OrigAuthEngineID string

	secParameters, ok := response.SecurityParameters.(*UsmSecurityParameters)
	if ok && secParameters != nil {
		OrigAuthEngineID = secParameters.AuthoritativeEngineID
	}
	response.Variables = make([]SnmpPDU, 0, 5)

	// Start parsing the packet
	cursor := 0

	// First bytes should be 0x30
	if PDUType(packet[0]) != Sequence {
		return fmt.Errorf("Invalid packet header\n")
	}

	length, cursor := parseLength(packet)
	if len(packet) != length {
		return fmt.Errorf("Error verifying packet sanity: Got %d Expected: %d\n", len(packet), length)
	}
	if x.loggingEnabled {
		x.Logger.Printf("Packet sanity verified, we got all the bytes (%d)", length)
	}

	// Parse SNMP Version
	rawVersion, count, err := x.parseRawField(packet[cursor:], "version")
	if err != nil {
		return fmt.Errorf("Error parsing SNMP packet version: %s", err.Error())
	}

	cursor += count
	if version, ok := rawVersion.(int); ok {
		response.Version = SnmpVersion(version)
		if x.loggingEnabled {
			x.Logger.Printf("Parsed version %d", version)
		}
	}
	if response.Version != Version3 {
		// Parse community
		rawCommunity, count, err := x.parseRawField(packet[cursor:], "community")
		if err != nil {
			return fmt.Errorf("Error parsing community string: %s", err.Error())
		}
		cursor += count
		if community, ok := rawCommunity.(string); ok {
			response.Community = community
			if x.loggingEnabled {
				x.Logger.Printf("Parsed community %s", community)
			}
		}
	} else {
		if PDUType(packet[cursor]) != Sequence {
			return fmt.Errorf("Invalid SNMPV3 Header\n")
		}

		_, cursorTmp := parseLength(packet[cursor:])
		cursor += cursorTmp

		rawMsgID, count, err := x.parseRawField(packet[cursor:], "msgID")
		if err != nil {
			return fmt.Errorf("Error parsing SNMPV3 message ID: %s", err.Error())
		}
		cursor += count
		if MsgID, ok := rawMsgID.(int); ok {
			response.MsgID = uint32(MsgID)
			if x.loggingEnabled {
				x.Logger.Printf("Parsed message ID %d", MsgID)

			}
		}
		// discard msg max size
		_, count, err = x.parseRawField(packet[cursor:], "maxMsgSize")
		if err != nil {
			return fmt.Errorf("Error parsing SNMPV3 maxMsgSize: %s", err.Error())
		}
		cursor += count
		// discard msg max size

		rawMsgFlags, count, err := x.parseRawField(packet[cursor:], "msgFlags")
		if err != nil {
			return fmt.Errorf("Error parsing SNMPV3 msgFlags: %s", err.Error())
		}
		cursor += count
		if MsgFlags, ok := rawMsgFlags.(string); ok {
			response.MsgFlags = SnmpV3MsgFlags(MsgFlags[0])
			if x.loggingEnabled {
				x.Logger.Printf("parsed msg flags %s", MsgFlags)
			}
		}

		rawSecModel, count, err := x.parseRawField(packet[cursor:], "msgSecurityModel")
		if err != nil {
			return fmt.Errorf("Error parsing SNMPV3 msgSecModel: %s", err.Error())
		}
		cursor += count
		if SecModel, ok := rawSecModel.(int); ok {
			response.SecurityModel = SnmpV3SecurityModel(SecModel)
			if x.loggingEnabled {
				x.Logger.Printf("Parsed security model %d", SecModel)
			}
		}

		if PDUType(packet[cursor]) != OctetString {
			return fmt.Errorf("Invalid SNMPV3 Security Parameters\n")
		}
		_, cursorTmp = parseLength(packet[cursor:])
		cursor += cursorTmp

		if response.SecurityModel == UserSecurityModel {
			secParameters, ok := response.SecurityParameters.(*UsmSecurityParameters)
			if !ok || secParameters == nil {
				return fmt.Errorf("&GoSNMP.SecurityModel indicates the User Security Model, but &GoSNMP.SecurityParameters is not of type &UsmSecurityParameters")
			}

			if PDUType(packet[cursor]) != Sequence {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model parameters\n")
			}
			_, cursorTmp = parseLength(packet[cursor:])
			cursor += cursorTmp

			rawMsgAuthoritativeEngineID, count, err := x.parseRawField(packet[cursor:], "msgAuthoritativeEngineID")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgAuthoritativeEngineID: %s", err.Error())
			}
			cursor += count
			if AuthoritativeEngineID, ok := rawMsgAuthoritativeEngineID.(string); ok {
				secParameters.AuthoritativeEngineID = AuthoritativeEngineID
				if x.loggingEnabled {
					x.Logger.Printf("Parsed authoritativeEngineID %s", AuthoritativeEngineID)
				}
			}

			rawMsgAuthoritativeEngineBoots, count, err := x.parseRawField(packet[cursor:], "msgAuthoritativeEngineBoots")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgAuthoritativeEngineBoots: %s", err.Error())
			}
			cursor += count
			if AuthoritativeEngineBoots, ok := rawMsgAuthoritativeEngineBoots.(int); ok {
				secParameters.AuthoritativeEngineBoots = uint32(AuthoritativeEngineBoots)
				if x.loggingEnabled {
					x.Logger.Printf("Parsed authoritativeEngineBoots %d", AuthoritativeEngineBoots)
				}
			}

			rawMsgAuthoritativeEngineTime, count, err := x.parseRawField(packet[cursor:], "msgAuthoritativeEngineTime")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgAuthoritativeEngineTime: %s", err.Error())
			}
			cursor += count
			if AuthoritativeEngineTime, ok := rawMsgAuthoritativeEngineTime.(int); ok {
				secParameters.AuthoritativeEngineTime = uint32(AuthoritativeEngineTime)
				if x.loggingEnabled {
					x.Logger.Printf("Parsed authoritativeEngineTime %d", AuthoritativeEngineTime)
				}
			}

			rawMsgUserName, count, err := x.parseRawField(packet[cursor:], "msgUserName")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgUserName: %s", err.Error())
			}
			cursor += count
			if msgUserName, ok := rawMsgUserName.(string); ok {
				secParameters.UserName = msgUserName
				if x.loggingEnabled {
					x.Logger.Printf("Parsed userName %s", msgUserName)
				}
			}

			rawMsgAuthParameters, count, err := x.parseRawField(packet[cursor:], "msgAuthenticationParameters")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgAuthenticationParameters: %s", err.Error())
			}

			if msgAuthenticationParameters, ok := rawMsgAuthParameters.(string); ok {
				secParameters.AuthenticationParameters = msgAuthenticationParameters
				if x.loggingEnabled {
					x.Logger.Printf("Parsed authenticationParameters %s", msgAuthenticationParameters)
				}
			}
			// use the authoritative copy of MsgFlags to determine whether this message should be authenticated
			if OrigMsgFlags&AuthNoPriv > 0 {
				if count != 14 {
					return fmt.Errorf("Error authenticating incoming packet: msgAuthenticationParameters is not the correct size")
				}
				blank := make([]byte, 12)
				copy(packet[cursor+2:cursor+14], blank)
				// secParameters.AuthenticationProtocol and secParameters.AuthenticationPassphrase are not written to in this function,
				// so no need to save an 'original' to authenticate against
				if !isAuthentic(packet, secParameters.AuthenticationParameters,
					secParameters.AuthenticationProtocol,
					secParameters.AuthenticationPassphrase,
					OrigAuthEngineID) {
					return fmt.Errorf("Incoming packet is not authentic, discarding")
				}
			}
			cursor += count

			rawMsgPrivacyParameters, count, err := x.parseRawField(packet[cursor:], "msgPrivacyParameters")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 User Security Model msgPrivacyParameters: %s", err.Error())
			}
			cursor += count
			if msgPrivacyParameters, ok := rawMsgPrivacyParameters.(string); ok {
				secParameters.PrivacyParameters = []byte(msgPrivacyParameters)
				if x.loggingEnabled {
					x.Logger.Printf("Parsed privacyParameters %s", msgPrivacyParameters)
				}
			}

			//response.SecurityParameters = &secParameters
		}
		switch PDUType(packet[cursor]) {
		case OctetString:
			// pdu is encrypted
			_, cursorTmp := parseLength(packet[cursor:])
			cursorTmp += cursor

			if response.SecurityModel == UserSecurityModel {
				var secParams *UsmSecurityParameters
				secParams, ok := response.SecurityParameters.(*UsmSecurityParameters)
				if !ok || secParams == nil {
					return fmt.Errorf("response.SecurityModel indicates the User Security Model, but response.SecurityParameters is not of type &UsmSecurityParameters")
				}
				var privkey = genlocalkey(secParams.AuthenticationProtocol,
					secParams.PrivacyPassphrase,
					secParams.AuthoritativeEngineID)
				switch secParams.PrivacyProtocol {
				case AES:
					var iv [16]byte
					binary.BigEndian.PutUint32(iv[:], secParams.AuthoritativeEngineBoots)
					binary.BigEndian.PutUint32(iv[4:], secParams.AuthoritativeEngineTime)
					copy(iv[8:], secParams.PrivacyParameters)

					block, err := aes.NewCipher(privkey[:16])
					if err != nil {
						return err
					}
					stream := cipher.NewCFBDecrypter(block, iv[:])
					plaintext := make([]byte, len(packet[cursorTmp:]))
					stream.XORKeyStream(plaintext, packet[cursorTmp:])
					copy(packet[cursor:], plaintext)
					packet = packet[:cursor+len(plaintext)]
				default:
					if len(packet[cursorTmp:])%des.BlockSize != 0 {
						return fmt.Errorf("Error decrypting ScopedPDU: not multiple of des block size.")
					}
					preiv := privkey[8:]
					var iv [8]byte
					for i := 0; i < len(iv); i++ {
						iv[i] = preiv[i] ^ secParams.PrivacyParameters[i]
					}
					block, err := des.NewCipher(privkey[:8])
					if err != nil {
						return err
					}
					mode := cipher.NewCBCDecrypter(block, iv[:])

					plaintext := make([]byte, len(packet[cursorTmp:]))
					mode.CryptBlocks(plaintext, packet[cursorTmp:])
					copy(packet[cursor:], plaintext)
					// truncate packet to remove extra space caused by the
					// octetstring/length header that was just replaced
					packet = packet[:cursor+len(plaintext)]
				}

			}
			fallthrough
		case Sequence:
			// pdu is plaintext
			tlength, cursorTmp := parseLength(packet[cursor:])
			// truncate padding that may have been included with
			// the encrypted PDU
			packet = packet[:cursor+tlength]
			cursor += cursorTmp
			rawContextEngineID, count, err := x.parseRawField(packet[cursor:], "contextEngineID")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 contextEngineID: %s", err.Error())
			}
			cursor += count
			if contextEngineID, ok := rawContextEngineID.(string); ok {
				response.ContextEngineID = contextEngineID
				if x.loggingEnabled {
					x.Logger.Printf("Parsed contextEngineID %s", contextEngineID)
				}
			}
			rawContextName, count, err := x.parseRawField(packet[cursor:], "contextName")
			if err != nil {
				return fmt.Errorf("Error parsing SNMPV3 contextName: %s", err.Error())
			}
			cursor += count
			if contextName, ok := rawContextName.(string); ok {
				response.ContextName = contextName
				if x.loggingEnabled {
					x.Logger.Printf("Parsed contextName %s", contextName)
				}
			}

		default:
			return fmt.Errorf("Error parsing SNMPV3 scoped PDU\n")
		}
	}
	// Parse SNMP packet type
	requestType := PDUType(packet[cursor])
	switch requestType {
	// known, supported types
	case GetResponse, GetNextRequest, GetBulkRequest, Report, SNMPv2Trap:
		response, err = x.unmarshalResponse(packet[cursor:], response, length, requestType)
		if err != nil {
			return fmt.Errorf("Error in unmarshalResponse: %s", err.Error())
		}
	default:
		return fmt.Errorf("Unknown PDUType %#x", requestType)
	}
	return nil
}

func (x *GoSNMP) unmarshalResponse(packet []byte, response *SnmpPacket, length int, requestType PDUType) (*SnmpPacket, error) {
	cursor := 0
	response.PDUType = requestType

	getResponseLength, cursor := parseLength(packet)
	if len(packet) != getResponseLength {
		return nil, fmt.Errorf("Error verifying Response sanity: Got %d Expected: %d\n", len(packet), getResponseLength)
	}
	if x.loggingEnabled {
		x.Logger.Printf("getResponseLength: %d", getResponseLength)
	}

	// Parse Request-ID
	rawRequestID, count, err := x.parseRawField(packet[cursor:], "request id")
	if err != nil {
		return nil, fmt.Errorf("Error parsing SNMP packet request ID: %s", err.Error())
	}
	cursor += count
	if requestid, ok := rawRequestID.(int); ok {
		response.RequestID = uint32(requestid)
		if x.loggingEnabled {
			x.Logger.Printf("requestID: %d", response.RequestID)
		}
	}

	if response.PDUType == GetBulkRequest {
		// Parse Non Repeaters
		rawNonRepeaters, count, err := x.parseRawField(packet[cursor:], "non repeaters")
		if err != nil {
			return nil, fmt.Errorf("Error parsing SNMP packet non repeaters: %s", err.Error())
		}
		cursor += count
		if nonRepeaters, ok := rawNonRepeaters.(int); ok {
			response.NonRepeaters = uint8(nonRepeaters)
		}

		// Parse Max Repetitions
		rawMaxRepetitions, count, err := x.parseRawField(packet[cursor:], "max repetitions")
		if err != nil {
			return nil, fmt.Errorf("Error parsing SNMP packet max repetitions: %s", err.Error())
		}
		cursor += count
		if maxRepetitions, ok := rawMaxRepetitions.(int); ok {
			response.MaxRepetitions = uint8(maxRepetitions)
		}
	} else {
		// Parse Error-Status
		rawError, count, err := x.parseRawField(packet[cursor:], "error-status")
		if err != nil {
			return nil, fmt.Errorf("Error parsing SNMP packet error: %s", err.Error())
		}
		cursor += count
		if errorStatus, ok := rawError.(int); ok {
			response.Error = SNMPError(errorStatus)
			if x.loggingEnabled {
				x.Logger.Printf("errorStatus: %d", uint8(errorStatus))
			}
		}

		// Parse Error-Index
		rawErrorIndex, count, err := x.parseRawField(packet[cursor:], "error index")
		if err != nil {
			return nil, fmt.Errorf("Error parsing SNMP packet error index: %s", err.Error())
		}
		cursor += count
		if errorindex, ok := rawErrorIndex.(int); ok {
			response.ErrorIndex = uint8(errorindex)
			if x.loggingEnabled {
				x.Logger.Printf("error-index: %d", uint8(errorindex))
			}
		}
	}

	return x.unmarshalVBL(packet[cursor:], response, length)
}

// unmarshal a Varbind list
func (x *GoSNMP) unmarshalVBL(packet []byte, response *SnmpPacket,
	length int) (*SnmpPacket, error) {

	var cursor, cursorInc int
	var vblLength int
	if packet[cursor] != 0x30 {
		return nil, fmt.Errorf("Expected a sequence when unmarshalling a VBL, got %x",
			packet[cursor])
	}

	vblLength, cursor = parseLength(packet)
	if len(packet) != vblLength {
		return nil, fmt.Errorf("Error verifying: packet length %d vbl length %d\n",
			len(packet), vblLength)
	}
	if x.loggingEnabled {
		x.Logger.Printf("vblLength: %d", vblLength)
	}

	// check for an empty response
	if vblLength == 2 && packet[1] == 0x00 {
		return response, nil
	}

	// Loop & parse Varbinds
	for cursor < vblLength {
		if packet[cursor] != 0x30 {
			return nil, fmt.Errorf("Expected a sequence when unmarshalling a VB, got %x", packet[cursor])
		}

		_, cursorInc = parseLength(packet[cursor:])
		cursor += cursorInc

		// Parse OID
		rawOid, oidLength, err := x.parseRawField(packet[cursor:], "OID")
		if err != nil {
			return nil, fmt.Errorf("Error parsing OID Value: %s", err.Error())
		}
		cursor += oidLength

		var oid []int
		var ok bool
		if oid, ok = rawOid.([]int); !ok {
			return nil, fmt.Errorf("unable to type assert rawOid |%v| to []int", rawOid)
		}
		oidStr := oidToString(oid)
		if x.loggingEnabled {
			x.Logger.Printf("OID: %s", oidStr)
		}

		// Parse Value
		v, err := x.decodeValue(packet[cursor:], "value")
		if err != nil {
			return nil, fmt.Errorf("Error decoding value: %v", err)
		}
		valueLength, _ := parseLength(packet[cursor:])
		cursor += valueLength
		response.Variables = append(response.Variables, SnmpPDU{oidStr, v.Type, v.Value, x.Logger})
	}
	return response, nil
}

// receive response from network and read into a byte array
func (x *GoSNMP) receive() ([]byte, error) {
	n, err := x.Conn.Read(x.rxBuf[:])
	if err != nil {
		return nil, fmt.Errorf("Error reading from UDP: %s", err.Error())
	}

	if n == rxBufSize {
		// This should never happen unless we're using something like a unix domain socket.
		return nil, fmt.Errorf("response buffer too small")
	}

	resp := make([]byte, n)
	copy(resp, x.rxBuf[:n])
	return resp, nil
}
