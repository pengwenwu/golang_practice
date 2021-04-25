package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	hash     Hash           // 函数类型，依赖注入，允许用于替换成自定义的Hash函数
	replicas int            // 虚拟节点倍数
	keys     []int          // Sorted 哈希环
	hashMap  map[int]string // 虚拟节点与真实节点的映射表，键：虚拟节点哈希值，值：真实节点名称
}

// New creates a Map instance
// 构造函数：允许自定义虚拟节点倍数和Hash函数
func New(relicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: relicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash
// Add 函数允许传入0或者多个真实节点的名称
// 对每一个真实节点key，对应创建m.replicas个虚拟节点，虚拟节点的名称通过添加编号的方式区分不同虚拟节点
// 使用m.hash()计算虚拟节点的哈希值，追加到环上
// hashMap中增加虚拟节点和真实节点的映射关系
// 最后，环上的哈希值排序
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key 在哈希环提供的key中查找最近的节点
// 计算key的哈希值
// 顺时针找到第一个匹配的虚拟节点的下标idx，从m.keys中获取到对应的哈希值。环状结构，取余
// 通过hashMap映射获取到真实的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica 二分查找
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
