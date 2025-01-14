package websocket

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gospider007/gson"
	"github.com/gospider007/tools"
	"github.com/gospider007/websocket/websocket"
)

type MessageType int

const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage MessageType = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage MessageType = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage MessageType = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage MessageType = 9

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage MessageType = 10
)

type Option struct {
	Subprotocol            string
	EnableCompression      bool
	ReadBufferSize         int
	WriteBufferSize        int
	NewCompressionWriter   func(io.WriteCloser, int) io.WriteCloser
	NewDecompressionReader func(io.Reader) io.ReadCloser
}

func SetClientHeadersWithOption(headers http.Header, option Option) {
	websocket.SetClientHeadersOption(headers, websocket.Option(option))
}

func GetResponseHeaderOption(header http.Header) Option {
	return Option(websocket.GetResponseHeaderOption(header))
}
func GetRequestHeaderOption(header http.Header) Option {
	return Option(websocket.GetRequestHeaderOption(header))
}

func NewClientConn(conn net.Conn, option Option) *Conn {
	con := websocket.NewClientConn(conn, websocket.Option(option))
	return &Conn{
		conn:   con,
		rawCon: conn,
	}
}
func NewServerConn(conn net.Conn, option Option) *Conn {
	con := websocket.NewServerConn(conn, websocket.Option(option))
	return &Conn{
		conn:     con,
		rawCon:   conn,
		IsServer: true,
	}
}

type UpgradeOption struct {
	HandshakeTimeout                time.Duration
	ReadBufferSize, WriteBufferSize int
	Subprotocols                    []string
	Error                           func(w http.ResponseWriter, r *http.Request, status int, reason error)
	CheckOrigin                     func(r *http.Request) bool
	EnableCompression               bool
}

func NewServerConnWithHTTP(w http.ResponseWriter, r *http.Request, responseHeader http.Header, options ...UpgradeOption) (*Conn, error) {
	var option UpgradeOption
	if len(options) > 0 {
		option = options[0]
	}
	up := websocket.Upgrader{
		HandshakeTimeout:  option.HandshakeTimeout,
		ReadBufferSize:    option.ReadBufferSize,
		WriteBufferSize:   option.WriteBufferSize,
		Subprotocols:      option.Subprotocols,
		Error:             option.Error,
		CheckOrigin:       option.CheckOrigin,
		EnableCompression: option.EnableCompression,
	}
	con, err := up.Upgrade(w, r, responseHeader)
	return &Conn{conn: con, IsServer: true}, err
}

type Conn struct {
	conn     *websocket.Conn
	rawCon   net.Conn
	rlock    sync.Mutex
	lock     sync.Mutex
	IsServer bool
}

func (obj *Conn) ReadMessage() (MessageType, []byte, error) {
	obj.rlock.Lock()
	defer obj.rlock.Unlock()
	mesg, con, err := obj.conn.ReadMessage()
	return MessageType(mesg), con, err
}
func (obj *Conn) Close() error {
	err := obj.conn.Close()
	if obj.rawCon != nil {
		obj.rawCon.Close()
	}
	return err
}

func (obj *Conn) WriteMessage(messageType MessageType, p any) error {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	switch val := p.(type) {
	case []byte:
		return obj.conn.WriteMessage(int(messageType), val)
	case string:
		return obj.conn.WriteMessage(int(messageType), tools.StringToBytes(val))
	default:
		con, err := gson.Encode(p)
		if err != nil {
			return err
		}
		return obj.conn.WriteMessage(int(messageType), con)
	}
}
