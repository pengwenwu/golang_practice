package gee

import (
	"net/http"
)

// HandlerFunc defines the request handler used by gee （定义处理方法）
type HandlerFunc func(ctx *Context)

// Engine implement the interface of ServeHTTP （实现对应接口）
type Engine struct {
	router *router
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}

// New is the constructor of gee.Engine (构造方法)
func New() *Engine {
	return &Engine{router: newRouter()}
}

func (engine *Engine) addRoute(method, pattern string, handler HandlerFunc) {
	engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request (定义GET方法)
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request (定义POST方法)
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// Run defines the method to start a http server （定义http启动方法)）
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}
