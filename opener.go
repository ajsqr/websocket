package websocket 

import (
	"crypto/sha1"
	"net/http"
	"encoding/base64"
	"bufio"
	"fmt"
)

const (
	websocketGUID =  "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

// Opener defines the methods possible by a websocket opener
type Opener interface {
	// Open will open a websocket connection, by upgrading the existing HTTP connection.
	// The opened websocket connection hijacks the existing http connection.
	Open(w http.ResponseWriter, r *http.Request, t WebsocketType) (Websocket, error)
}

type WSOpener struct {
	// MaxBytes defines the maximum payload length of a frame.
	// If the message is bigger than this value, then the message is sent as fragments.
	MaxBytes int
}

// Open will open a websocket connection, by upgrading the existing HTTP connection.
// The opened websocket connection hijacks the existing http connection.
func (wso *WSOpener) Open(w http.ResponseWriter, r *http.Request, t WebsocketType) (*Websocket, error) {
	ws := Websocket{}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, HijackingNotSupported
	}

	// the second return item here is a ReadWriter 
	// I should probably reuse this instead of creating a new one.
	conn, _, err := hj.Hijack()
	if err != nil{
		return nil, err
	}

	ws.reader = bufio.NewReader(conn)
	ws.writer = bufio.NewWriter(conn)
	ws.t = t
	ws.framingLimit = wso.MaxBytes

	err = wso.handshake(ws.writer, r)
	if err != nil{
		return nil, err
	}

	return &ws, nil
}

// handshake performs the websocket handshake
func (wso *WSOpener) handshake(writer *bufio.Writer,r *http.Request) error {
	websocketKey := r.Header.Get("Sec-WebSocket-Key")
	acceptToken := generateWebsocketAcceptToken(websocketKey)
	response := newWebsocketAcceptResponse(acceptToken)
	err := response.Write(writer)
	if err != nil{
		return err
	}

	err = writer.Flush()
	if err != nil{
		return err
	}

	return nil
}

// generateWebsocketAcceptToken generates the token used as 
// Sec-WebSocket-Accept header field.  The value of this
// header field is constructed by concatenating /key/, defined
// above in step 4 in Section 4.2.2, with the string "258EAFA5-
// E914-47DA-95CA-C5AB0DC85B11", taking the SHA-1 hash of this
// concatenated value to obtain a 20-byte value and base64-
// encoding (see Section 4 of [RFC4648]) this 20-byte hash.
func generateWebsocketAcceptToken(secWebsocketKey string) string {
	combinedKey := []byte(fmt.Sprintf("%s%s", secWebsocketKey, websocketGUID))
	hash := sha1.Sum(combinedKey)
	encodedKey := base64.StdEncoding.EncodeToString(hash[:])
	return encodedKey
}

func newWebsocketAcceptResponse(acceptToken string) *http.Response {
	resp := http.Response{
		Status: "101 Switching Protocols",
		StatusCode: 101,
		Header: http.Header{},
		Proto: "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	resp.Header.Set("Upgrade", "websocket")
	resp.Header.Set("Connection", "Upgrade")
	resp.Header.Set("Sec-WebSocket-Accept", acceptToken)
	return &resp
} 