package websocket 

import (
	"errors"
)

var (
	
	BadRequest = errors.New("bad request")

	InvalidFrameType = errors.New("invalid frame type")

	HijackingNotSupported = errors.New("response writer implementation does not support hijacking")

	InvalidOpcode  = errors.New("invalid opcode")

	InvalidLength  = errors.New("invalid length")

)