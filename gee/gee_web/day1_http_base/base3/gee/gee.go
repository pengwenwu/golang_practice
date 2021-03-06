package gee

import (
	"fmt"
	"net/http"
)

// HandlerFunc defines the request handler used by gee （定义处理方法）
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// Engine implement the interface of ServeHTTP （实现对应接口）
type Engine struct {
	router map[string]HandlerFunc
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}

// New is the constructor of gee.Engine (构造方法)
func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}

func (engine *Engine) addRoute(method, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
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
