package geekcache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"

	"geekcache/consistenthash"
	pb "geekcache/geecachepb"
)

const defaultBasePath = "/_geecache/" // 节点间通讯地址前缀
const defaultReplicas = 50

// HTTPPool implements PeerPicker for a pool of HTTP peers
// 新增成员变量peers，类型是一致性哈希算法的map，用来根据具体的key选择节点
// 新增成员变量httpGetters，映射远程节点与对应的httpGetter。每一个远程节点对应一个httpGetter，因为httpGetter与远程节点的地址baseUrl有关
type HTTPPool struct {
	// this peer's base URL, e.g. "https://expamle.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving expected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the value to the response body as a proto message
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// Set updates the pool's list of peers
// Set() 方法实例化了一致性哈希算法，并且添加了传入的节点，并为每一个节点创建了一个HTTP客户端httpGetter
func (h *HTTPPool) Set(peers ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.peers = consistenthash.New(defaultReplicas, nil)
	h.peers.Add(peers...)
	h.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		h.httpGetters[peer] = &httpGetter{
			baseUrl: peer + h.basePath,
		}
	}
}

// PickPerr picks a peer according to key
// 包装了一致性哈希算法的Get()方法，根据具体的key选择节点，返回节点对应的HTTP客户端
func (h *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if peer := h.peers.Get(key); peer != "" && peer != h.self {
		h.Log("Pick peer %s", peer)
		return h.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// --http客户端--
type httpGetter struct {
	baseUrl string
}

// baseUrl 表示将要访问的远程节点的地址，例如：http://examle.com/_geecache/
// 使用http.Get() 方式获取返回值，并转换为[]bytes类型
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf("%v%v/%v", h.baseUrl, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)
