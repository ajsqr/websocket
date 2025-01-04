package websocket

import (
	"io"
	"bufio"
	"math"
	"bytes"
	"context"
	"encoding/binary"
)

type WebsockerServer interface {
	// Send transports the message from the server to the the client.
	Send(ctx context.Context, message []byte) error 

	// Receive waits for a message from the client.
	Receive(ctx context.Context) ([]byte, error)

	// Close closes an active websocket connection between the client and the server.
	Close(ctx context.Context) error
}

type WebsocketType string

var (
	TextWebsocket WebsocketType = "text"

	BinaryWebsocket WebsocketType = "binary"
)

type Websocket struct {
	reader *bufio.Reader 
	writer *bufio.Writer
	t WebsocketType 
	framingLimit int
}

func (ws *Websocket) Close() error {
	return nil
}

// Send transports the message from the server to the the client.
func (ws *Websocket) Send(ctx context.Context, data []byte) error {
	frames, err := ws.fragment(ctx, data)
	if err != nil{
		return err
	}

	for _, frame := range frames {
		err := ws.writeFrame(frame)
		if err != nil{
			return err
		}

	}

	return nil
}

// Receive waits for a message from the client.
func (ws *Websocket) Receive(ctx context.Context) ([]byte, error) {
	message := make([]byte, 0)
	for {
		frame, err := ws.readFrame()
		if err != nil{
			return message, err
		}

		umasked, err := frame.umask()
		if err != nil{
			return message, err
		}

		for _, um := range umasked{
			message = append(message, um)
		}
		
		if frame.FIN {
			break
		}
	}
	
	return message, nil

}


// readFrame reads a single frame from the response stream.
func (ws *Websocket) readFrame() (*Frame, error){
	f := Frame{}
	// The first byte contains a lot of metadata.
	// |FIN |RSV1|RSV2|RSV3|     OPCODE     |
	// We have to selectively read the bits to parse the metadata.
	b, err := ws.reader.ReadByte()
	if err != nil{
		return nil, err
	}

	if b&0x80 == 0x80{
		// this selectively sets all except the MSB to 0
		f.FIN = true
	}

	// we are ignoring RSV1 - RSV3 for now :(

	opcode := b&0x0f
	switch(opcode){
	case 0x00:
		f.Opcode = ContinuationFrame 
	case 0x01:
		f.Opcode = TextFrame
	case 0x02:
		f.Opcode = BinaryFrame 
	case 0x03, 0x04, 0x05, 0x06, 0x07:
		f.Opcode = NonControlFrame 
	case 0x08:
		f.Opcode = ConnectionClose
	case 0x09:
		f.Opcode = Ping 
	case 0x0a:
		f.Opcode = Pong 
	case 0x0b, 0x0c, 0x0d, 0x0e, 0x0f:
		f.Opcode =  ControlFrame
	default:
		return nil, InvalidOpcode
	}

	payloadMetaData, err := ws.reader.ReadByte()
	if err != nil{
		return nil, err
	}

	if payloadMetaData&0x80 == 0x80 {
		// see if the mask bit is set
		f.Mask = true
	}

	payloadLengthMetadata := payloadMetaData&0x7f

	if 0<= payloadLengthMetadata && payloadLengthMetadata <= 125{
		s := uint(payloadLengthMetadata)
		f.payloadLengthInt = &s
	} else if payloadLengthMetadata == 126{
		// the next two bytes is the length
		length := make([]byte, 2)
		_, err := ws.reader.Read(length)
		if err != nil{
			return nil, err
		}
		s := binary.BigEndian.Uint16(length)
		f.payloadLengthInt16 = &s
	} else if payloadLengthMetadata == 127 {
		// the next four bytes is the length
		length := make([]byte, 8)
		_, err := ws.reader.Read(length)
		if err != nil{
			return nil, err
		}

		s := binary.BigEndian.Uint64(length)
		f.payloadLengthInt64 = &s
	} else {
		return nil, InvalidLength
	}

	if f.Mask {
		// we infer that the frame is masked
		maskingKey := make([]byte, 4)
		_, err := ws.reader.Read(maskingKey)
		if err != nil{
			return nil, BadRequest
		}

		f.MaskingKey = maskingKey
	}

	// we assume here that there are no extensions
	payload := make([]byte, f.PayloadLength())
	_, err = ws.reader.Read(payload)
		if err != nil{
			return nil, BadRequest
		}

		f.ApplicationData = payload


	return &f, nil

}

func (ws *Websocket) writeFrame(frame *Frame) error {
	frameIdentifier := 0x00 // a byte with all the bits unset
	if frame.FIN {
		// set the first bit to zero if the frame FIN is true
		frameIdentifier |= 0x80 
	}

	// ignoring all RSV bits for now
	switch(frame.Opcode){
	// the last 4 bits of the first byte has the opcode
	// depending on the opcode of the frame, we have to selectively set 
	// the last four bits
	case ContinuationFrame:
		frameIdentifier |= 0x00
	case TextFrame:
		frameIdentifier |= 0x01
	case BinaryFrame:
		frameIdentifier |= 0x02 
	case ConnectionClose:
		frameIdentifier |= 0x08
	case Ping:
		frameIdentifier |= 0x09
	case Pong:
		frameIdentifier |= 0x0A 
	default:
		return InvalidOpcode
	}

	err := ws.writer.WriteByte(byte(frameIdentifier))
	if err != nil{
		return err
	}

	var payloadLength uint64  // masking is not required for frames from server
	if 0 <= frame.PayloadLength() && frame.PayloadLength ()<= 125 {
		// binLength := binary.LittleEndian.PutUint32
		payloadLength = frame.PayloadLength()
	} else if frame.PayloadLength() <= math.MaxUint16 {
		// If 126, the following 2 bytes interpreted as a
		// 16-bit unsigned integer are the payload length
		payloadLength = 126
	} else if frame.PayloadLength() <= math.MaxUint64 {
		// If 127, the following 8 bytes interpreted as a 64-bit 
		// unsigned integer (the most significant bit MUST be 0) are the payload length
		payloadLength = 127
	} else {
		return InvalidLength
	}

	err = ws.writer.WriteByte(byte(payloadLength))
	if err != nil{
		return err
	}

	if payloadLength == 126 {
		// write the lenth as Uint16 (2 bytes)
		payloadLength := make([]byte, 2)
		binary.BigEndian.PutUint16(payloadLength, *frame.payloadLengthInt16)
		_, err := ws.writer.Write(payloadLength)
		if err != nil{
			return err
		}

	} else if payloadLength == 127 {
		// write the lenth as Uint64 (8 bytes)
		payloadLength := make([]byte, 8)
		binary.BigEndian.PutUint64(payloadLength, *frame.payloadLengthInt64)
		_, err := ws.writer.Write(payloadLength)
		if err != nil{
			return err
		}
	} 

	_, err = ws.writer.Write(frame.ApplicationData)
	if err != nil{
		return err
	}

	err = ws.writer.Flush()
	if err != nil{
		return err
	}

	return nil

}


// fragment will fragment the payload based on the fragmentation settings
func (ws *Websocket) fragment(ctx context.Context, data []byte) ([]*Frame, error) {
	frames := make([]*Frame, 0)
	fragmentReader := bytes.NewReader(data)
	payloadLength := len(data)
	for {
		chunkSize := payloadLength
		if ws.framingLimit < chunkSize{
			chunkSize = ws.framingLimit
		}

		chunk := make([]byte, chunkSize)
		read, err := fragmentReader.Read(chunk)
		
		if err == io.EOF {
			break
		}

		frame := Frame{
			ApplicationData: chunk, 
			Opcode: ContinuationFrame,
		}

		chunkLength := len(chunk)
		if chunkLength < 126{
			convertedLength := uint(chunkLength)
			frame.payloadLengthInt = &convertedLength
		} else if uint16(chunkLength) >= 126 && uint16(chunkLength) < math.MaxUint16{
			convertedLength := uint16(chunkLength)
			frame.payloadLengthInt16 = &convertedLength
		} else if uint64(chunkLength) >= math.MaxUint16 && uint64(chunkLength) < math.MaxUint64{
			convertedLength := uint64(chunkLength)
			frame.payloadLengthInt64 = &convertedLength
		} else{
			return frames, InvalidLength
		}

		frames = append(frames, &frame)

		payloadLength -= read
	}

	if len(frames) > 0 {
		// set the fin flag for the last frame 
		frames[len(frames)-1].FIN = true

		switch(ws.t) {
		case TextWebsocket:
			frames[0].Opcode = TextFrame 
		case BinaryWebsocket:
			frames[0].Opcode = BinaryFrame
		default:
			return frames, InvalidFrameType
		}

	}

	return frames, nil
}

