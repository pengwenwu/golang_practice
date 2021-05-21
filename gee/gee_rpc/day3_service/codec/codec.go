package codec

import "io"

// ServiceMethod 是服务名和方法名，通常与Go语言中的结构体和方法相映射
// Seq 是请求的序号，也可以认为是某个请求的ID，用来区分不同的请求
// Error 是错误信息，客户端置为空，服务端如果发生错误，将错误信息置于Error中
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init()  {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}

