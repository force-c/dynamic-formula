package dynamicformula

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// KeyActualF 节点 Key 常量 前置原始节点
const (
	KeyActualF         = "actual_f"         // 实际分时电费
	KeyTotalF          = "total_f"          // 合计当期电费
	KeyLongTermF       = "longterm_f"       // 中长期市场化电费
	KeyDADevF          = "dadev_f"          // 日前偏差电费
	KeyRTDevF          = "rtdev_f"          // 实时偏差电费
	KeyTransferF       = "transfer_f"       // 偏差价差收益转移电费
	KeyOriginalF       = "original_f"       // 原电费
	KeyDeviationSettle = "deviation_settle" // 偏差结算费用
)

// MomentData 原始"时刻颗粒"数据容器
type MomentData struct {
	Period                          int
	ActualQ, ActualP, ActualF       float64
	TotalQ, TotalP, TotalF          float64
	LongTermQ, LongTermP, LongTermF float64
	DADevQ, DADevP, DADevF          float64
	RTDevQ, RTDevP, RTDevF          float64
	TransferQ, TransferP, TransferF float64
}

// Result 计算节点输出值
type Result struct {
	Key   string
	Value float64
}

type Node interface {
	Name() string
	Requires() []string
	Compute(m MomentData, prev map[string]Result) (Result, error)
}

// baseNode 内置前置节点
type baseNode struct {
	name string
	f    func(MomentData) (q, p, f float64)
}

func (b baseNode) Name() string       { return b.name }
func (b baseNode) Requires() []string { return nil }
func (b baseNode) Compute(m MomentData, _ map[string]Result) (Result, error) {
	_, _, f := b.f(m)
	return Result{Key: b.name, Value: f}, nil
}

// FormulaNode 公式节点
type FormulaNode struct {
	name    string
	deps    []string
	formula func(m MomentData, prev map[string]Result) float64
}

func (f FormulaNode) Name() string       { return f.name }
func (f FormulaNode) Requires() []string { return f.deps }
func (f FormulaNode) Compute(m MomentData, prev map[string]Result) (Result, error) {
	return Result{Key: f.name, Value: f.formula(m, prev)}, nil
}

// cacheEntry 缓存条目，包含值和过期时间
type cacheEntry struct {
	value  interface{}
	expiry time.Time
}

// TTLCache 自实现 TTL 缓存，使用 sync.Map + 定时器清理
type TTLCache struct {
	data      sync.Map
	mu        sync.RWMutex
	cleanupCh chan struct{}
	done      chan struct{}
}

// NewTTLCache 创建 TTL 缓存实例
func NewTTLCache() *TTLCache {
	c := &TTLCache{
		cleanupCh: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	c.startCleanup(time.Minute)
	return c
}

// Set 存储键值对，设置 TTL
func (c *TTLCache) Set(key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("invalid TTL: %v", ttl)
	}
	entry := &cacheEntry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
	c.data.Store(key, entry)
	c.triggerCleanup()
	return nil
}

// Get 获取值，过期返回 false
func (c *TTLCache) Get(key string) (interface{}, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := val.(*cacheEntry)
	if !ok {
		c.data.Delete(key)
		return nil, false
	}
	if time.Now().After(entry.expiry) {
		c.data.Delete(key)
		return nil, false
	}
	return entry.value, true
}

// Delete 删除指定键
func (c *TTLCache) Delete(key string) error {
	c.data.Delete(key)
	return nil
}

// triggerCleanup 触发清理检查
func (c *TTLCache) triggerCleanup() {
	select {
	case c.cleanupCh <- struct{}{}:
	default:
	}
}

// cleanupLoop 后台清理过期条目
func (c *TTLCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.cleanupCh:
			c.cleanupExpired()
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.done:
			return
		}
	}
}

// cleanupExpired 清理所有过期条目
func (c *TTLCache) cleanupExpired() {
	now := time.Now()
	c.data.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*cacheEntry); ok && now.After(entry.expiry) {
			c.data.Delete(key)
		}
		return true
	})
}

// startCleanup 启动后台清理 goroutine
func (c *TTLCache) startCleanup(interval time.Duration) {
	go c.cleanupLoop(interval)
}

// Stop 停止缓存，清理资源
func (c *TTLCache) Stop() {
	close(c.done)
	c.data.Range(func(key, _ interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

var (
	baseRegistry map[string]Node
	sortCache    *TTLCache
	cacheTTL     = 5 * time.Minute
)

func init() {
	// 注册 gob 类型（用于测试）
	gob.Register([]Node{})
	gob.Register([]string{})
	gob.Register(Result{})
	gob.Register(FormulaNode{})
	gob.Register(baseNode{})

	baseRegistry = make(map[string]Node)
	RegisterBase(baseNode{name: KeyActualF, f: func(m MomentData) (float64, float64, float64) {
		return m.ActualQ, m.ActualP, m.ActualF
	}})
	RegisterBase(baseNode{name: KeyTotalF, f: func(m MomentData) (float64, float64, float64) {
		return m.TotalQ, m.TotalP, m.TotalF
	}})
	RegisterBase(baseNode{name: KeyLongTermF, f: func(m MomentData) (float64, float64, float64) {
		return m.LongTermQ, m.LongTermP, m.LongTermF
	}})
	RegisterBase(baseNode{name: KeyDADevF, f: func(m MomentData) (float64, float64, float64) {
		return m.DADevQ, m.DADevP, m.DADevF
	}})
	RegisterBase(baseNode{name: KeyRTDevF, f: func(m MomentData) (float64, float64, float64) {
		return m.RTDevQ, m.RTDevP, m.RTDevF
	}})
	RegisterBase(baseNode{name: KeyTransferF, f: func(m MomentData) (float64, float64, float64) {
		return m.TransferQ, m.TransferP, m.TransferF
	}})

	sortCache = NewTTLCache()
}

// RegisterBase 注册基础节点
func RegisterBase(n Node) {
	if _, ok := baseRegistry[n.Name()]; ok {
		panic("duplicate base node: " + n.Name())
	}
	baseRegistry[n.Name()] = n
}

// CalcTemplate 封装任务节点配置
type CalcTemplate struct {
	registry map[string]Node
	mu       sync.RWMutex
}

// NewCalcTemplate 创建模板，复制基础节点并添加衍生节点
func NewCalcTemplate(extraNodes ...Node) *CalcTemplate {
	t := &CalcTemplate{registry: make(map[string]Node)}
	t.mu.Lock()
	defer t.mu.Unlock()

	for k, v := range baseRegistry {
		t.registry[k] = v
	}
	for _, n := range extraNodes {
		if _, ok := t.registry[n.Name()]; ok {
			panic("duplicate node in template: " + n.Name())
		}
		t.registry[n.Name()] = n
	}
	return t
}

// GetHash 生成模板唯一 hash
func (t *CalcTemplate) GetHash() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var keys []string
	for k := range t.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hashStr := ""
	for _, k := range keys {
		n := t.registry[k]
		deps := n.Requires()
		sort.Strings(deps)
		hashStr += fmt.Sprintf("%s:%s;", k, deps)
	}

	h := sha256.New()
	h.Write([]byte(hashStr))
	return hex.EncodeToString(h.Sum(nil))
}

// GetOrderedNodes 从缓存获取或计算拓扑排序
func (t *CalcTemplate) GetOrderedNodes() ([]Node, error) {
	hash := t.GetHash()

	if val, found := sortCache.Get(hash); found {
		return val.([]Node), nil
	}

	ordered, err := t.topoSortTemplate()
	if err != nil {
		return nil, err
	}

	if err := sortCache.Set(hash, ordered, cacheTTL); err != nil {
		return nil, fmt.Errorf("cache set failed: %w", err)
	}
	return ordered, nil
}

// topoSortTemplate 基于模板的拓扑排序
func (t *CalcTemplate) topoSortTemplate() ([]Node, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	inDegree := make(map[string]int, len(t.registry))
	graph := make(map[string][]string, len(t.registry))
	for name := range t.registry {
		inDegree[name] = 0
		graph[name] = nil
	}
	for _, n := range t.registry {
		for _, dep := range n.Requires() {
			if _, ok := t.registry[dep]; !ok {
				return nil, fmt.Errorf("unknown dependency %s -> %s", n.Name(), dep)
			}
			graph[dep] = append(graph[dep], n.Name())
			inDegree[n.Name()]++
		}
	}
	var queue []string
	// 按名称排序入度为 0 的节点，确保稳定顺序
	var zeroInDegree []string
	for name, d := range inDegree {
		if d == 0 {
			zeroInDegree = append(zeroInDegree, name)
		}
	}
	sort.Strings(zeroInDegree) // 稳定排序
	queue = append(queue, zeroInDegree...)

	var sorted []Node
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		sorted = append(sorted, t.registry[cur])
		// 按名称排序邻接节点，确保稳定顺序
		var nextZero []string
		for _, next := range graph[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				nextZero = append(nextZero, next)
			}
		}
		if len(nextZero) > 0 {
			sort.Strings(nextZero)
			queue = append(queue, nextZero...)
		}
	}
	if len(sorted) != len(t.registry) {
		return nil, fmt.Errorf("cycle detected")
	}
	return sorted, nil
}

// Calc MomentData 的计算方法
func (m MomentData) Calc(t *CalcTemplate) (map[string]Result, error) {
	ordered, err := t.GetOrderedNodes()
	if err != nil {
		return nil, err
	}
	done := make(map[string]Result, len(ordered))
	for _, n := range ordered {
		res, err := n.Compute(m, done)
		if err != nil {
			return nil, fmt.Errorf("node %s compute failed: %w", n.Name(), err)
		}
		done[n.Name()] = res
	}
	return done, nil
}

// CalcBatch 批量并行计算
func CalcBatch(dataList []MomentData, templates []*CalcTemplate) ([]map[string]Result, []error) {
	if len(dataList) != len(templates) {
		return nil, []error{fmt.Errorf("dataList and templates length mismatch")}
	}
	results := make([]map[string]Result, len(dataList))
	errors := make([]error, len(dataList))
	var wg sync.WaitGroup

	for i, m := range dataList {
		wg.Add(1)
		go func(i int, m MomentData, t *CalcTemplate) {
			defer wg.Done()
			res, err := m.Calc(t)
			results[i] = res
			errors[i] = err
		}(i, m, templates[i])
	}
	wg.Wait()

	return results, errors
}
