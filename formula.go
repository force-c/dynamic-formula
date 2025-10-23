package dynamicformula

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/force-c/dynamic-formula/utils"
)

// OptionalFloat 是 float64 的别名
type OptionalFloat float64

// NewOptionalFloat 创建 *OptionalFloat
func NewOptionalFloat(f float64) *OptionalFloat {
	v := OptionalFloat(f)
	return &v
}

// Result 包含 Q, P, F 三个字段
type Result struct {
	Q *OptionalFloat
	P *OptionalFloat
	F *OptionalFloat
}

// MomentData 表示输入数据
type MomentData struct {
	Period    int            // 时刻
	ActualQ   *OptionalFloat // 实际分时电量
	ActualP   *OptionalFloat // 实际分时电价
	ActualF   *OptionalFloat // 实际分时电费
	TotalQ    *OptionalFloat // 合计当期电量
	TotalP    *OptionalFloat // 合计当期电价
	TotalF    *OptionalFloat // 合计当期电费
	LongTermQ *OptionalFloat // 中长期市场变化电量
	LongTermP *OptionalFloat // 中长期市场变化电价
	LongTermF *OptionalFloat // 中长期市场变化电费
	DADevQ    *OptionalFloat // 日前偏差电量
	DADevP    *OptionalFloat // 日前偏差电价
	DADevF    *OptionalFloat // 日前偏差电费
	RTDevQ    *OptionalFloat // 实时偏差电量
	RTDevP    *OptionalFloat // 实时偏差电价
	RTDevF    *OptionalFloat // 实时偏差电费
	TransferQ *OptionalFloat // 传输电量
	TransferP *OptionalFloat // 传输电价
	TransferF *OptionalFloat // 传输电费
}

// Node 是计算节点的接口
type Node interface {
	Name() string
	Requires() []string
	Compute(m MomentData, done map[string]interface{}) (interface{}, error)
}

// baseNode 是基础节点
type baseNode struct {
	name string
	f    func(MomentData) (q, p, f *OptionalFloat) // 返回 *OptionalFloat，无 error
}

func (n baseNode) Name() string {
	return n.name
}

func (n baseNode) Requires() []string {
	return nil
}

func (n baseNode) Compute(m MomentData, _ map[string]interface{}) (interface{}, error) {
	q, p, f := n.f(m)
	return Result{Q: q, P: p, F: f}, nil
}

// FormulaNode 是公式节点
type FormulaNode struct {
	name    string
	deps    []string
	formula func(MomentData, map[string]interface{}) (float64, error)
}

func (n FormulaNode) Name() string {
	return n.name
}

func (n FormulaNode) Requires() []string {
	return n.deps
}

func (n FormulaNode) Compute(m MomentData, done map[string]interface{}) (interface{}, error) {
	return n.formula(m, done)
}

// CalcTemplate 是计算模板
type CalcTemplate struct {
	nodes    []Node
	registry map[string]Node // 存储节点及其依赖
}

// NewCalcTemplate 创建新模板，收集依赖节点
func NewCalcTemplate(nodes ...Node) *CalcTemplate {
	t := &CalcTemplate{
		nodes:    nodes,
		registry: make(map[string]Node),
	}

	// 收集所有依赖节点
	required := make(map[string]bool)
	for _, n := range nodes {
		collectDependencies(n, required)
		t.registry[n.Name()] = n
	}

	// 只添加依赖的基础节点和公式节点
	for name := range required {
		if node, ok := baseRegistry[name]; ok {
			t.registry[name] = node
		} else if node, ok := formulaRegistry[name]; ok {
			t.registry[name] = node
		} else {
			panic("unknown dependency: " + name)
		}
	}

	return t
}

// collectDependencies 递归收集节点及其依赖
func collectDependencies(n Node, required map[string]bool) {
	if required[n.Name()] {
		return
	}
	required[n.Name()] = true
	for _, dep := range n.Requires() {
		if node, ok := baseRegistry[dep]; ok {
			collectDependencies(node, required)
		} else if node, ok := formulaRegistry[dep]; ok {
			collectDependencies(node, required)
		} else {
			panic("unknown dependency: " + dep)
		}
	}
}

// NewFullCalcTemplate 包含所有计算节点
func NewFullCalcTemplate() *CalcTemplate {
	return NewCalcTemplate(
		formulaRegistry[KeyOriginalF],
		formulaRegistry[KeyDeviationSettle],
		formulaRegistry[KeyDeviationProfit],
		formulaRegistry[KeyTotalFee],
		formulaRegistry[KeyFinalProfit],
		formulaRegistry[KeyArbitrage],
	)
}

// GetOrderedNodes 返回拓扑排序后的节点列表
func (t *CalcTemplate) GetOrderedNodes() ([]Node, error) {
	cacheKey := ""
	for _, n := range t.nodes {
		cacheKey += n.Name() + ";"
	}

	if cached, ok := sortCache.Get(cacheKey); ok {
		return cached.([]Node), nil
	}

	visited := make(map[string]bool)
	temp := make(map[string]bool)
	result := make([]Node, 0, len(t.nodes))
	var dfs func(n Node) error

	dfs = func(n Node) error {
		if temp[n.Name()] {
			return fmt.Errorf("cycle detected at node %s", n.Name())
		}
		if visited[n.Name()] {
			return nil
		}
		temp[n.Name()] = true
		for _, dep := range n.Requires() {
			var next Node
			if node, ok := t.registry[dep]; ok {
				next = node
			} else if node, ok := formulaRegistry[dep]; ok {
				next = node
			} else if node, ok := baseRegistry[dep]; ok {
				next = node
			} else {
				return fmt.Errorf("node %s not found", dep)
			}
			if err := dfs(next); err != nil {
				return err
			}
		}
		temp[n.Name()] = false
		visited[n.Name()] = true
		result = append(result, n)
		return nil
	}

	for _, n := range t.nodes {
		if !visited[n.Name()] {
			if err := dfs(n); err != nil {
				return nil, err
			}
		}
	}

	sortCache.Set(cacheKey, result, time.Hour)
	return result, nil
}

// Calc 执行计算
func (m MomentData) Calc(t *CalcTemplate, includeBaseNodes bool) (map[string]interface{}, error) {
	ordered, err := t.GetOrderedNodes()
	if err != nil {
		return nil, err
	}
	done := make(map[string]interface{}, len(ordered))
	results := make(map[string]interface{})
	for _, n := range ordered {
		res, err := n.Compute(m, done)
		if err != nil {
			return nil, fmt.Errorf("node %s compute failed: %w", n.Name(), err)
		}
		done[n.Name()] = res
		if includeBaseNodes || reflect.TypeOf(n) == reflect.TypeOf(FormulaNode{}) {
			// 格式化输出
			if result, ok := res.(Result); ok {
				// 基础节点：将 Result 格式化为 {Q, P, F} 字符串
				q := "<nil>"
				p := "<nil>"
				f := "<nil>"
				if result.Q != nil {
					q = fmt.Sprintf("%v", float64(*result.Q))
				}
				if result.P != nil {
					p = fmt.Sprintf("%v", float64(*result.P))
				}
				if result.F != nil {
					f = fmt.Sprintf("%v", float64(*result.F))
				}
				results[n.Name()] = fmt.Sprintf("{%s, %s, %s}", q, p, f)
			} else {
				// 计算节点：直接存储 float64
				results[n.Name()] = res
			}
		}
	}
	return results, nil
}

// TTLCache 是带过期时间的缓存
type TTLCache struct {
	cache map[string]cacheEntry
	mutex sync.RWMutex
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewTTLCache 创建新缓存
func NewTTLCache() *TTLCache {
	return &TTLCache{
		cache: make(map[string]cacheEntry),
	}
}

// Set 设置缓存
func (c *TTLCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Get 获取缓存
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiration) {
		return nil, false
	}
	return entry.value, true
}

// 常量定义
const (
	KeyActualF         = "实际分时电费"    // 实际分时电费
	KeyTotalF          = "合计当期电费"    // 合计当期电费
	KeyLongTermF       = "中长期市场变化电费" // 中长期市场变化电费
	KeyDADevF          = "日前偏差电费"    // 日前偏差电费
	KeyRTDevF          = "实时偏差电费"    // 实时偏差电费
	KeyTransferF       = "传输电费"      // 传输电费
	KeyOriginalF       = "原电费"       // 原电费
	KeyDeviationSettle = "偏差结算费用"    // 偏差结算费用
	KeyDeviationProfit = "偏差收益"      // 偏差收益
	KeyTotalFee        = "总电费"       // 总电费
	KeyFinalProfit     = "最终收益"      // 最终收益
	KeyArbitrage       = "套利收益"      // 套利收益
)

// 注册表
var (
	formulaRegistry map[string]Node
	baseRegistry    map[string]Node
	registryMutex   sync.RWMutex
	sortCache       *TTLCache
)

func RegisterBase(n baseNode) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	if baseRegistry == nil {
		baseRegistry = make(map[string]Node)
	}
	baseRegistry[n.name] = n
}

func RegisterFormula(n FormulaNode) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	if formulaRegistry == nil {
		formulaRegistry = make(map[string]Node)
	}
	formulaRegistry[n.name] = n
}

func init() {
	// 初始化注册表
	baseRegistry = make(map[string]Node)
	formulaRegistry = make(map[string]Node)
	sortCache = NewTTLCache()

	// 注册基础节点
	RegisterBase(baseNode{
		name: KeyActualF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.ActualQ, m.ActualP, m.ActualF
		},
	})
	RegisterBase(baseNode{
		name: KeyTotalF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.TotalQ, m.TotalP, m.TotalF
		},
	})
	RegisterBase(baseNode{
		name: KeyLongTermF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.LongTermQ, m.LongTermP, m.LongTermF
		},
	})
	RegisterBase(baseNode{
		name: KeyDADevF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.DADevQ, m.DADevP, m.DADevF
		},
	})
	RegisterBase(baseNode{
		name: KeyRTDevF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.RTDevQ, m.RTDevP, m.RTDevF
		},
	})
	RegisterBase(baseNode{
		name: KeyTransferF,
		f: func(m MomentData) (q, p, f *OptionalFloat) {
			return m.TransferQ, m.TransferP, m.TransferF
		},
	})

	// 注册公式节点
	// 原电费 = 中长期市场变化电费 + 日前偏差电费 + 实时偏差电费
	RegisterFormula(FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyLongTermF, KeyDADevF, KeyRTDevF},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			if m.LongTermF == nil {
				return 0, fmt.Errorf("field LongTermF is not set")
			}
			if m.DADevF == nil {
				return 0, fmt.Errorf("field DADevF is not set")
			}
			if m.RTDevF == nil {
				return 0, fmt.Errorf("field RTDevF is not set")
			}

			longTermF, ok := prev[KeyLongTermF].(Result)
			if !ok || longTermF.F == nil {
				return 0, fmt.Errorf("invalid LongTermF data")
			}
			daDevF, ok := prev[KeyDADevF].(Result)
			if !ok || daDevF.F == nil {
				return 0, fmt.Errorf("invalid DADevF data")
			}
			rtDevF, ok := prev[KeyRTDevF].(Result)
			if !ok || rtDevF.F == nil {
				return 0, fmt.Errorf("invalid RTDevF data")
			}
			//fmt.Printf("原电费计算 1 > %v 2 > %v 3 > %v", float64(*longTermF.F), float64(*daDevF.F), float64(*rtDevF.F))
			fmt.Println()
			// 使用 utils.DecimalAdd 进行高精度加法
			return utils.DecimalAdd(float64(*longTermF.F), float64(*daDevF.F), float64(*rtDevF.F)), nil
		},
	})

	// 偏差结算费用：
	// if 日前偏差电价 < 实时偏差电价:
	//   (合计当期电量 - 中长期市场变化电量 - 日前偏差电量) * (日前偏差电价 - 实时偏差电价)
	// else:
	//   (中长期市场变化电量 + 日前偏差电量 - 实际分时电量) * (实时偏差电价 - 日前偏差电价)
	RegisterFormula(FormulaNode{
		name: KeyDeviationSettle,
		deps: []string{KeyTotalF, KeyLongTermF, KeyDADevF, KeyRTDevF, KeyActualF},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			if m.DADevP == nil || m.RTDevP == nil || m.DADevQ == nil || m.LongTermQ == nil || m.TotalQ == nil || m.ActualQ == nil {
				return 0, fmt.Errorf("required fields are not set")
			}

			totalQ, ok := prev[KeyTotalF].(Result)
			if !ok || totalQ.Q == nil {
				return 0, fmt.Errorf("invalid TotalF data")
			}
			longTermQ, ok := prev[KeyLongTermF].(Result)
			if !ok || longTermQ.Q == nil {
				return 0, fmt.Errorf("invalid LongTermF data")
			}
			daDevQ, ok := prev[KeyDADevF].(Result)
			if !ok || daDevQ.Q == nil {
				return 0, fmt.Errorf("invalid DADevF data")
			}
			actualQ, ok := prev[KeyActualF].(Result)
			if !ok || actualQ.Q == nil {
				return 0, fmt.Errorf("invalid ActualF data")
			}
			daDevP, ok := prev[KeyDADevF].(Result)
			if !ok || daDevP.P == nil {
				return 0, fmt.Errorf("invalid DADevP data")
			}
			rtDevP, ok := prev[KeyRTDevF].(Result)
			if !ok || rtDevP.P == nil {
				return 0, fmt.Errorf("invalid RTDevP data")
			}

			// 使用 utils.Decimal* 函数进行高精度计算
			var result float64
			if float64(*daDevP.P) < float64(*rtDevP.P) {
				// (totalQ - longTermQ - daDevQ) * (daDevP - rtDevP)
				diffQ := utils.DecimalSubtract(float64(*totalQ.Q), float64(*longTermQ.Q))
				diffQ = utils.DecimalSubtract(diffQ, float64(*daDevQ.Q))
				diffP := utils.DecimalSubtract(float64(*daDevP.P), float64(*rtDevP.P))
				result = utils.DecimalMul(diffQ, diffP)
			} else {
				// (longTermQ + daDevQ - actualQ) * (rtDevP - daDevP)
				sumQ := utils.DecimalAdd(float64(*longTermQ.Q), float64(*daDevQ.Q))
				sumQ = utils.DecimalSubtract(sumQ, float64(*actualQ.Q))
				diffP := utils.DecimalSubtract(float64(*rtDevP.P), float64(*daDevP.P))
				result = utils.DecimalMul(sumQ, diffP)
			}

			return result, nil
		},
	})

	// 偏差收益：
	// if 日前偏差电价 < 实时偏差电价:
	//   (日前偏差电量 + 中长期市场变化电量 - 实际分时电量 * 1.2) * (实时偏差电价 - 日前偏差电价)
	// else:
	//   (合计当期电量 * 0.8 - 中长期市场变化电量 - 日前偏差电量) * (日前偏差电价 - 实时偏差电价)
	RegisterFormula(FormulaNode{
		name: KeyDeviationProfit,
		deps: []string{KeyTotalF, KeyLongTermF, KeyDADevF, KeyRTDevF, KeyActualF},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			if m.DADevP == nil || m.RTDevP == nil || m.DADevQ == nil || m.LongTermQ == nil || m.TotalQ == nil || m.ActualQ == nil {
				return 0, fmt.Errorf("required fields are not set")
			}

			totalQ, ok := prev[KeyTotalF].(Result)
			if !ok || totalQ.Q == nil {
				return 0, fmt.Errorf("invalid TotalF data")
			}
			longTermQ, ok := prev[KeyLongTermF].(Result)
			if !ok || longTermQ.Q == nil {
				return 0, fmt.Errorf("invalid LongTermF data")
			}
			daDevQ, ok := prev[KeyDADevF].(Result)
			if !ok || daDevQ.Q == nil {
				return 0, fmt.Errorf("invalid DADevF data")
			}
			actualQ, ok := prev[KeyActualF].(Result)
			if !ok || actualQ.Q == nil {
				return 0, fmt.Errorf("invalid ActualF data")
			}
			daDevP, ok := prev[KeyDADevF].(Result)
			if !ok || daDevP.P == nil {
				return 0, fmt.Errorf("invalid DADevP data")
			}
			rtDevP, ok := prev[KeyRTDevF].(Result)
			if !ok || rtDevP.P == nil {
				return 0, fmt.Errorf("invalid RTDevP data")
			}

			// 使用 utils.Decimal* 函数进行高精度计算
			var result float64
			if float64(*daDevP.P) < float64(*rtDevP.P) {
				//   (日前偏差电量 + 中长期市场变化电量 - 实际分时电量 * 1.2) * (实时偏差电价 - 日前偏差电价)
				//(daDevQ + longTermQ - actualQ * 1.2) * (rtDevP - daDevP)
				actualQMul := utils.DecimalMul(float64(*actualQ.Q), 1.2)
				sumQ := utils.DecimalAdd(float64(*daDevQ.Q), float64(*longTermQ.Q))
				sumQ = utils.DecimalSubtract(sumQ, actualQMul)
				diffP := utils.DecimalSubtract(float64(*rtDevP.P), float64(*daDevP.P))
				result = utils.DecimalMul(sumQ, diffP)
				//// Step 1: 计算 actualQMul = actualQ.Q * 1.2
				//fmt.Printf("[Step1] actualQ.Q = %v\n", *actualQ.Q)
				//actualQMul := utils.DecimalMul(float64(*actualQ.Q), 1.2)
				//fmt.Printf("[Step1] actualQMul = actualQ.Q * 1.2 = %v\n\n", actualQMul)
				//
				//// Step 2: sumQ = daDevQ.Q + longTermQ.Q
				//fmt.Printf("[Step2] daDevQ.Q = %v, longTermQ.Q = %v\n", *daDevQ.Q, *longTermQ.Q)
				//sumQ := utils.DecimalAdd(float64(*daDevQ.Q), float64(*longTermQ.Q))
				//fmt.Printf("[Step2] sumQ(before subtract) = daDevQ.Q + longTermQ.Q = %v\n\n", sumQ)
				//
				//// Step 3: sumQ = sumQ - actualQMul
				//fmt.Printf("[Step3] sumQ(before) = %v, actualQMul = %v\n", sumQ, actualQMul)
				//sumQ = utils.DecimalSubtract(sumQ, actualQMul)
				//fmt.Printf("[Step3] sumQ(after subtract) = sumQ - actualQMul = %v\n\n", sumQ)
				//
				//// Step 4: diffP = rtDevP.P - daDevP.P
				//fmt.Printf("[Step4] rtDevP.P = %v, daDevP.P = %v\n", *rtDevP.P, *daDevP.P)
				//diffP := utils.DecimalSubtract(float64(*rtDevP.P), float64(*daDevP.P))
				//fmt.Printf("[Step4] diffP = rtDevP.P - daDevP.P = %v\n\n", diffP)
				//
				//// Step 5: result = sumQ * diffP
				//fmt.Printf("[Step5] sumQ = %v, diffP = %v\n", sumQ, diffP)
				//result = utils.DecimalMul(sumQ, diffP)
				//fmt.Printf("[Step5] result = sumQ * diffP = %v\n", result)
			} else {
				// (totalQ * 0.8 - longTermQ - daDevQ) * (daDevP - rtDevP)
				totalQMul := utils.DecimalMul(float64(*totalQ.Q), 0.8)
				diffQ := utils.DecimalSubtract(totalQMul, float64(*longTermQ.Q))
				diffQ = utils.DecimalSubtract(diffQ, float64(*daDevQ.Q))
				diffP := utils.DecimalSubtract(float64(*daDevP.P), float64(*rtDevP.P))
				result = utils.DecimalMul(diffQ, diffP)
			}

			return result, nil
		},
	})

	// 总电费 = 原电费 + 偏差收益
	RegisterFormula(FormulaNode{
		name: KeyTotalFee,
		deps: []string{KeyOriginalF, KeyDeviationProfit},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			originalF, ok := prev[KeyOriginalF].(float64)
			if !ok {
				return 0, fmt.Errorf("KeyOriginalF not found or invalid type")
			}
			// 修正：使用 KeyDeviationSettle 而非 KeyDeviationProfit
			devSettle, ok := prev[KeyDeviationProfit].(float64)
			if !ok {
				return 0, fmt.Errorf("KeyDeviationSettle not found or invalid type")
			}

			// 使用 utils.DecimalAdd 进行高精度加法
			return utils.DecimalAdd(originalF, devSettle), nil
		},
	})

	// 最终收益 = 偏差结算费用 - 偏差收益
	RegisterFormula(FormulaNode{
		name: KeyFinalProfit,
		deps: []string{KeyDeviationSettle, KeyDeviationProfit},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			devSettle, ok := prev[KeyDeviationSettle].(float64)
			if !ok {
				return 0, fmt.Errorf("KeyDeviationSettle not found or invalid type")
			}
			devProfit, ok := prev[KeyDeviationProfit].(float64)
			if !ok {
				return 0, fmt.Errorf("KeyDeviationProfit not found or invalid type")
			}

			// 使用 utils.DecimalSubtract 进行高精度减法
			return utils.DecimalSubtract(devSettle, devProfit), nil
		},
	})

	// 套利收益 = 最终收益 / 合计当期电量
	RegisterFormula(FormulaNode{
		name: KeyArbitrage,
		deps: []string{KeyFinalProfit, KeyTotalF},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			if m.TotalQ == nil {
				return 0, fmt.Errorf("field TotalQ is not set")
			}
			totalQ, ok := prev[KeyTotalF].(Result)
			if !ok || totalQ.Q == nil {
				return 0, fmt.Errorf("invalid TotalF data")
			}
			finalProfit, ok := prev[KeyFinalProfit].(float64)
			if !ok {
				return 0, fmt.Errorf("KeyFinalProfit not found or invalid type")
			}

			if float64(*totalQ.Q) == 0 {
				return 0, nil // 避免除以 0
			}

			// 使用 utils.DecimalDivide 进行高精度除法，保留 4 位小数
			return utils.DecimalDivide(finalProfit, float64(*totalQ.Q), 4), nil
		},
	})
}
