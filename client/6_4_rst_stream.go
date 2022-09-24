package client

import (
	"golang.org/x/net/http2"

	"github.com/summerwind/h2spec/config"
	"github.com/summerwind/h2spec/spec"
)

func RSTStream() *spec.ClientTestGroup {
	tg := NewTestGroup("6.4", "RST_STREAM")

	// RST_STREAM frames MUST be associated with a stream.  If a
	// RST_STREAM frame is received with a stream identifier of 0x0,
	// the recipient MUST treat this as a connection error
	// (Section 5.4.1) of type PROTOCOL_ERROR.
	tg.AddTestCase(&spec.ClientTestCase{
		Desc:        "Sends a RST_STREAM frame with 0x0 stream identifier",
		Requirement: "The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		Run: func(c *config.ClientSpecConfig, conn *spec.Conn, req *spec.Request) error {
			conn.WriteRSTStream(0, http2.ErrCodeCancel)

			return spec.VerifyConnectionError(conn, http2.ErrCodeProtocol)
		},
	})

	// RST_STREAM frames MUST NOT be sent for a stream in the "idle"
	// state. If a RST_STREAM frame identifying an idle stream is
	// received, the recipient MUST treat this as a connection error
	// (Section 5.4.1) of type PROTOCOL_ERROR.
	tg.AddTestCase(&spec.ClientTestCase{
		Desc:        "Sends a RST_STREAM frame on a idle stream",
		Requirement: "The endpoint MUST respond with a connection error of type PROTOCOL_ERROR.",
		Run: func(c *config.ClientSpecConfig, conn *spec.Conn, req *spec.Request) error {
			conn.WriteRSTStream(2, http2.ErrCodeCancel)

			return spec.VerifyConnectionError(conn, http2.ErrCodeProtocol)
		},
	})

	// A RST_STREAM frame with a length other than 4 octets MUST be
	// treated as a connection error (Section 5.4.1) of type
	// FRAME_SIZE_ERROR.
	tg.AddTestCase(&spec.ClientTestCase{
		Desc:        "Sends a RST_STREAM frame with a length other than 4 octets",
		Requirement: "The endpoint MUST respond with a connection error of type FRAME_SIZE_ERROR.",
		Run: func(c *config.ClientSpecConfig, conn *spec.Conn, req *spec.Request) error {
			headers := spec.CommonRespHeaders(c)
			hp := http2.HeadersFrameParam{
				StreamID:      req.StreamID,
				EndStream:     true,
				EndHeaders:    true,
				BlockFragment: conn.EncodeHeaders(headers),
			}
			conn.WriteHeaders(hp)

			// RST_STREAM frame:
			// length: 3, flags: 0x0, stream_id: 0x01
			var flags http2.Flags
			conn.WriteRawFrame(http2.FrameRSTStream, flags, req.StreamID, []byte("\x00\x00\x00"))

			return spec.VerifyStreamError(conn, http2.ErrCodeFrameSize)
		},
	})

	return tg
}
