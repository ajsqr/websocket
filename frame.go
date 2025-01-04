package websocket 

// Defines the interpretation of the "Payload data".  If an unknown
// opcode is received, the receiving endpoint MUST _Fail the
// WebSocket Connection_.  The following values are defined.
type Opcode string 

var (
	// ContinuationFrame is the opcode send for fragments.
	// If the message length is greater than the fragmentationLimit of the opener,
	// the subsequent frames of the message will carry this opcode.
	ContinuationFrame Opcode = "continuation frame"

	// TextFrame is the opcode sent for text fragments.
	// In case of fragmented text messages, the first frame will carry this opcode.
	TextFrame Opcode = "text frame"

	// BinaryFrame is the opcode sent for binary fragments.
	// In case of fragmented binary messages, the first frame will carry this opcode.
	BinaryFrame Opcode = "binary frame"

	NonControlFrame Opcode = "non control frame"

	ConnectionClose Opcode = "connection close"

	Ping Opcode = "ping"

	Pong Opcode = "pong"

	ControlFrame Opcode = "control frame"
)

type Frame struct {
	// Indicates that this is the final fragment in a message.  
	// The first fragment MAY also be the final fragment.
	FIN bool 

	// MUST be 0 unless an extension is negotiated that defines meanings
	// for non-zero values.  If a nonzero value is received and none of
	// the negotiated extensions defines the meaning of such a nonzero
	// value, the receiving endpoint MUST _Fail the WebSocket
	// Connection_.
	RSV1 bool 

	RSV2 bool 

	RSV3 bool 

	// Opcode Defines the interpretation of the "Payload data".  If an unknown
	// opcode is received, the receiving endpoint MUST _Fail the
	// WebSocket Connection_.
	Opcode Opcode

	// Defines whether the "Payload data" is masked.  If set to 1, a
	// masking key is present in masking-key, and this is used to unmask
	// the "Payload data" as per Section 5.3.  All frames sent from
	// client to server have this bit set to 1
	Mask bool

	// The length of the "Payload data", in bytes: if 0-125, that is the
	// payload length.  If 126, the following 2 bytes interpreted as a
	// 16-bit unsigned integer are the payload length.  If 127, the
	// following 8 bytes interpreted as a 64-bit unsigned integer (the
	// most significant bit MUST be 0) are the payload length.  Multibyte
	// length quantities are expressed in network byte order.  Note that
	// in all cases, the minimal number of bytes MUST be used to encode
	// the length, for example, the length of a 124-byte-long string
	// can't be encoded as the sequence 126, 0, 124.
	//
	//  The payload length
	// is the length of the "Extension data" + the length of the
	// "Application data".  The length of the "Extension data" may be
	// zero, in which case the payload length is the length of the
	// "Application data".
	payloadLengthInt *uint
	payloadLengthInt16 *uint16
	payloadLengthInt64 *uint64


	// All frames sent from the client to the server are masked by a
	// 32-bit value that is contained within the frame.  This field is
	// present if the mask bit is set to 1 and is absent if the mask bit
	// is set to 0.  See Section 5.3 for further information on client-
	// to-server masking.
	MaskingKey []byte 

	// The "Extension data" is 0 bytes unless an extension has been
	// negotiated.  Any extension MUST specify the length of the
	// "Extension data", or how that length may be calculated, and how
	// the extension use MUST be negotiated during the opening handshake.
	// If present, the "Extension data" is included in the total payload
	// length.
	ExtensionData []byte 

	// Arbitrary "Application data", taking up the remainder of the frame
	// after any "Extension data".  The length of the "Application data"
	// is equal to the payload length minus the length of the "Extension
	// data".
	ApplicationData []byte
}

// PayloadLength gives the length of the payload in a frame.
// This method always returns the uint64 type
func (f *Frame) PayloadLength() uint64 {
	if f.payloadLengthInt != nil{
		return uint64(*f.payloadLengthInt)
	}

	if f.payloadLengthInt16 != nil{
		return uint64(*f.payloadLengthInt16)
	}

	if f.payloadLengthInt64 != nil{
		return uint64(*f.payloadLengthInt64)
	}

	return 0
}

// umask will decode the frame using the mask associated with it.
func (f *Frame) umask() ([]byte, error) {
	payload := f.ApplicationData
	umasked := make([]byte, len(payload))
	mask := f.MaskingKey 
	for i, b := range payload{
		j := i % 4
		umasked = append(umasked,b^mask[j])
	}

	return umasked, nil
}
