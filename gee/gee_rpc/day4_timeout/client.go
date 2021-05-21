package geerpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Call represents an active RPC
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

// Client represents an RPC Client
// There may be multiple outstanding Calls associated with a single Client, and a Client may be
// used by multiple goroutines simultaneously
// cc 是消息的编解码器，和服务端类似，用来序列化将要发送出去的请求，以及反序列化接收到的响应
// sending 是一个互斥锁，和服务端类似，为了保证请求的有序发送，即防止多个请求报文混淆
// header 是每个请求的消息头，header只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，申明在 Client 结构体中复用
// seq 用于给发送的请求编号，每个请求拥有唯一编号
// pending 存储未处理完的请求，键是编号，值是 Call 实例
// closing 和 shutdown 任意一个值置为true，则表示 client 处于不可用的状态，但有些许的差别，closing是用户主动关闭的，即调用 Close 方法，而shutdown置为true一般是有错误发生
type Client struct {
	cc       codec.Codec
	opt      *Option
	sending  sync.Mutex // protect following
	header   codec.Header
	mu       sync.Mutex // protecting following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closing {
		return ErrShutdown
	}
	c.closing = true
	return c.cc.Close()
}

// IsAvailable return true if the client does work
func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.shutdown && !c.closing
}

// registerCall 将参数 call 添加到 client.pending 中，并更新 client.seq.
func (c *Client) registerCall(call *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closing || c.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = c.seq
	c.pending[call.Seq] = call
	c.seq++
	return call.Seq, nil
}

// removeCall 根据 seq，从 client.pending 中移除对应的 call，并返回。
func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.pending[seq]
	delete(c.pending, seq)
	return call
}

// terminateCalls 服务端或客户端发生错误调用，将shutdown设置为true，且将错误信息通知所有pending状态的call
func (c *Client) terminateCalls(err error) {
	c.sending.Lock()
	defer c.sending.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdown = true
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}

// 对于一个客户端来说，接收响应、发送请求是最重要的2个功能。那么首先实现接收功能，接收到的响应有三种情况：
// - call不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
// - call存在，但是服务端处理出错，即h.Error不为空
// - call存在，服务端处理正常，那么需要从body中读取Reply的值
func (c *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = c.cc.ReadHeader(&h); err != nil {
			break
		}
		call := c.removeCall(h.Seq)
		switch {
		case call == nil:
			// it usually means that Write partially failed and call was already removed
			err = c.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = c.cc.ReadBody(nil)
			call.done()
		default:
			err = c.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// error occurs, so terminateCalls pending calls
	c.terminateCalls(err)
}

// 创建 Client 实例时，首先需要完成一开始的协议交换，即发送 Option 信息给服务端。
// 协商好消息的编解码方式之后，再创建一个子协程调用 receive() 接收响应
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	// send options with server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		cc:      cc,
		opt:     opt,
		seq:     1, // seq starts with 1, 0 means invalid call
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// 实现 Dial 函数，便于用户传入服务端地址，创建 Client 实例。为了简化用户调用，通过...*Option将Option实现为可选参数
// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	return dialTimeout(NewClient, network, address, opts...)
}

func parseOptions(opts ...*Option) (*Option, error) {
	// if opts is nil or pass nil as parameter
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

func (c *Client) send(call *Call) {
	// make sure that the client will send a complete request
	c.sending.Lock()
	defer c.sending.Unlock()

	// register this call.
	seq, err := c.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// prepare request header
	c.header.ServiceMethod = call.ServiceMethod
	c.header.Seq = seq
	c.header.Error = ""

	// encode and send the request
	if err := c.cc.Write(&c.header, call.Args); err != nil {
		call := c.removeCall(seq)
		// call may be nil, it usually means that Write partially failed,
		// client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go 和 Call 是客户端暴露给用户的两个RPC服务调用接口， Go 是一个异步接口，返回call实例。
// Call 是对 Go 的封装，阻塞call.Done，等待响应返回，是一个同步接口。
// Go invokes the function asynchronously
// It returns the Call structure repressing the invocation
func (c *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	c.send(call)
	return call
}

// Call invokes the named function, waits for it to complete
// and returns its error status
// Client.Call 的超时处理机制，使用 context 包实现，控制权交给用户，控制更为灵活
// 用户可以使用 context.WithTimeout 创建具备超时检测能力的 context 对象来控制。
func (c *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := c.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		c.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

// 这里实现了一个超时处理的外壳 dialTimeout ，这个壳将 NewClient 作为入参，在2个地方添加了超时处理的机制。
// - 将 net.Dial 替换为 net.DialTimeout ，如果连接创建超时，将返回错误
// - 使用子协程执行 NewClient ，执行完成后则通过信道 ch 发送结果，如果 time.After() 信道先接收到消息，则说明 NewClient 执行超时，返回错误。
func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address, opt.ConnectionTimeout)
	if err != nil {
		return nil, err
	}
	// close the connection if client is nil
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{
			client: client,
			err:    err,
		}
	}()
	if opt.ConnectionTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	case <-time.After(opt.ConnectionTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectionTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}
