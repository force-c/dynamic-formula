package dynamicformula

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/force-c/dynamic-formula/utils"
)

// OptionalFloat 用于包装 float64，可表示缺失值语义。
type OptionalFloat float64

// NewOptionalFloat 返回 OptionalFloat 指针。
func NewOptionalFloat(f float64) *OptionalFloat {
	v := OptionalFloat(f)
	return &v
}

// Result 封装输入节点产出的 Q/P/V 三元结果。
type Result struct {
	Q *OptionalFloat
	P *OptionalFloat
	V *OptionalFloat
}

// ContextInput 表示单次计算上下文，字段可按场景自由组合。
type ContextInput struct {
	Period int

	ObservedQ *OptionalFloat
	ObservedP *OptionalFloat
	ObservedV *OptionalFloat

	AggregateQ *OptionalFloat
	AggregateP *OptionalFloat
	AggregateV *OptionalFloat

	BaselineQ *OptionalFloat
	BaselineP *OptionalFloat
	BaselineV *OptionalFloat

	ScenarioAQ *OptionalFloat
	ScenarioAP *OptionalFloat
	ScenarioAV *OptionalFloat

	ScenarioBQ *OptionalFloat
	ScenarioBP *OptionalFloat
	ScenarioBV *OptionalFloat

	OverheadQ *OptionalFloat
	OverheadP *OptionalFloat
	OverheadV *OptionalFloat
}

// Node 表示计算图中的节点。
type Node interface {
	Name() string
	Requires() []string
	Compute(ContextInput, map[string]interface{}) (interface{}, error)
}

// InputAdapter 将上下文数据转换为标准 Result 结果。
type InputAdapter func(ContextInput) (q, p, v *OptionalFloat)

type inputNode struct {
	name    string
	resolve InputAdapter
}

func (n inputNode) Name() string { return n.name }

func (n inputNode) Requires() []string { return nil }

func (n inputNode) Compute(m ContextInput, _ map[string]interface{}) (interface{}, error) {
	q, p, v := n.resolve(m)
	return Result{Q: q, P: p, V: v}, nil
}

// FormulaNode 代表执行自定义公式的计算节点。
type FormulaNode struct {
	name    string
	deps    []string
	formula func(ContextInput, map[string]interface{}) (float64, error)
}

func (n FormulaNode) Name() string { return n.name }

func (n FormulaNode) Requires() []string { return n.deps }

func (n FormulaNode) Compute(m ContextInput, done map[string]interface{}) (interface{}, error) {
	return n.formula(m, done)
}

// CalcTemplate 保存选定节点与依赖关系。
type CalcTemplate struct {
	nodes    []Node
	registry map[string]Node
}

// NewCalcTemplate 根据传入节点收集依赖。
func NewCalcTemplate(nodes ...Node) *CalcTemplate {
	t := &CalcTemplate{
		nodes:    nodes,
		registry: make(map[string]Node),
	}

	required := make(map[string]bool)
	for _, n := range nodes {
		collectDependencies(n, required)
		t.registry[n.Name()] = n
	}

	for name := range required {
		if node, ok := inputRegistry[name]; ok {
			t.registry[name] = node
		} else if node, ok := formulaRegistry[name]; ok {
			t.registry[name] = node
		} else {
			panic("unknown dependency: " + name)
		}
	}

	return t
}

// collectDependencies 递归遍历依赖图。
func collectDependencies(n Node, required map[string]bool) {
	if required[n.Name()] {
		return
	}
	required[n.Name()] = true
	for _, dep := range n.Requires() {
		if node, ok := inputRegistry[dep]; ok {
			collectDependencies(node, required)
		} else if node, ok := formulaRegistry[dep]; ok {
			collectDependencies(node, required)
		} else {
			panic("unknown dependency: " + dep)
		}
	}
}

// NewFullCalcTemplate 返回包含所有默认公式的模板。
func NewFullCalcTemplate() *CalcTemplate {
	return NewCalcTemplate(
		formulaRegistry[KeyBaseCost],
		formulaRegistry[KeySettlementImpact],
		formulaRegistry[KeyScenarioMargin],
		formulaRegistry[KeyTotalCost],
		formulaRegistry[KeyNetMargin],
		formulaRegistry[KeyUnitYield],
	)
}

// GetOrderedNodes 以依赖顺序返回节点。
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
	var dfs func(Node) error

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
			} else if node, ok := inputRegistry[dep]; ok {
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

// Calc 在给定上下文中执行模板。
func (m ContextInput) Calc(t *CalcTemplate, includeInputNodes bool) (map[string]interface{}, error) {
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
		if includeInputNodes || reflect.TypeOf(n) == reflect.TypeOf(FormulaNode{}) {
			if result, ok := res.(Result); ok {
				q := "<nil>"
				p := "<nil>"
				v := "<nil>"
				if result.Q != nil {
					q = fmt.Sprintf("%v", float64(*result.Q))
				}
				if result.P != nil {
					p = fmt.Sprintf("%v", float64(*result.P))
				}
				if result.V != nil {
					v = fmt.Sprintf("%v", float64(*result.V))
				}
				results[n.Name()] = fmt.Sprintf("{%s, %s, %s}", q, p, v)
			} else {
				results[n.Name()] = res
			}
		}
	}
	return results, nil
}

// TTLCache 是带过期机制的内存缓存。
type TTLCache struct {
	cache map[string]cacheEntry
	mutex sync.RWMutex
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewTTLCache 创建缓存实例。
func NewTTLCache() *TTLCache {
	return &TTLCache{
		cache: make(map[string]cacheEntry),
	}
}

// Set 写入带 TTL 的缓存。
func (c *TTLCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Get 读取未过期的缓存值。
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiration) {
		return nil, false
	}
	return entry.value, true
}

const (
	// 默认输入节点标识符。
	KeyObservedMetrics   = "observed_metrics"
	KeyAggregateMetrics  = "aggregate_metrics"
	KeyBaselineMetrics   = "baseline_metrics"
	KeyScenarioAInputs   = "scenario_a_inputs"
	KeyScenarioBInputs   = "scenario_b_inputs"
	KeyOverheadAdjusters = "overhead_adjusters"
	// 默认公式节点标识符。
	KeyBaseCost         = "base_cost"
	KeySettlementImpact = "settlement_impact"
	KeyScenarioMargin   = "scenario_margin"
	KeyTotalCost        = "total_cost"
	KeyNetMargin        = "net_margin"
	KeyUnitYield        = "unit_yield"
)

var (
	formulaRegistry map[string]Node
	inputRegistry   map[string]Node
	registryMutex   sync.RWMutex
	sortCache       *TTLCache
)

// RegisterInputNode 注册自定义输入适配器。
func RegisterInputNode(name string, adapter InputAdapter) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	if inputRegistry == nil {
		inputRegistry = make(map[string]Node)
	}
	inputRegistry[name] = inputNode{
		name:    name,
		resolve: adapter,
	}
}

// RegisterInputAdapter 是 RegisterInputNode 的同义接口，更强调适配语义。
func RegisterInputAdapter(name string, adapter InputAdapter) {
	RegisterInputNode(name, adapter)
}

// RegisterFormula 将公式节点写入全局注册表。
func RegisterFormula(n FormulaNode) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	if formulaRegistry == nil {
		formulaRegistry = make(map[string]Node)
	}
	formulaRegistry[n.name] = n
}

func init() {
	inputRegistry = make(map[string]Node)
	formulaRegistry = make(map[string]Node)
	sortCache = NewTTLCache()

	RegisterInputNode(KeyObservedMetrics, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.ObservedQ, m.ObservedP, m.ObservedV
	})
	RegisterInputNode(KeyAggregateMetrics, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.AggregateQ, m.AggregateP, m.AggregateV
	})
	RegisterInputNode(KeyBaselineMetrics, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.BaselineQ, m.BaselineP, m.BaselineV
	})
	RegisterInputNode(KeyScenarioAInputs, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.ScenarioAQ, m.ScenarioAP, m.ScenarioAV
	})
	RegisterInputNode(KeyScenarioBInputs, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.ScenarioBQ, m.ScenarioBP, m.ScenarioBV
	})
	RegisterInputNode(KeyOverheadAdjusters, func(m ContextInput) (q, p, v *OptionalFloat) {
		return m.OverheadQ, m.OverheadP, m.OverheadV
	})

	// 基础成本 = 基线 + 场景 A + 场景 B 的估值。
	RegisterFormula(FormulaNode{
		name: KeyBaseCost,
		deps: []string{KeyBaselineMetrics, KeyScenarioAInputs, KeyScenarioBInputs},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			if m.BaselineV == nil {
				return 0, fmt.Errorf("context.BaselineV is nil")
			}
			if m.ScenarioAV == nil {
				return 0, fmt.Errorf("context.ScenarioAV is nil")
			}
			if m.ScenarioBV == nil {
				return 0, fmt.Errorf("context.ScenarioBV is nil")
			}

			baseline, ok := prev[KeyBaselineMetrics].(Result)
			if !ok || baseline.V == nil {
				return 0, fmt.Errorf("invalid baseline metrics data")
			}
			scenarioA, ok := prev[KeyScenarioAInputs].(Result)
			if !ok || scenarioA.V == nil {
				return 0, fmt.Errorf("invalid scenario A data")
			}
			scenarioB, ok := prev[KeyScenarioBInputs].(Result)
			if !ok || scenarioB.V == nil {
				return 0, fmt.Errorf("invalid scenario B data")
			}

			return utils.DecimalAdd(
				float64(*baseline.V),
				float64(*scenarioA.V),
				float64(*scenarioB.V),
			), nil
		},
	})

	// 结算影响：根据场景估算量、观测量与价格差。
	RegisterFormula(FormulaNode{
		name: KeySettlementImpact,
		deps: []string{
			KeyAggregateMetrics,
			KeyBaselineMetrics,
			KeyScenarioAInputs,
			KeyScenarioBInputs,
			KeyObservedMetrics,
		},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			if m.ScenarioAP == nil || m.ScenarioBP == nil || m.ScenarioAQ == nil || m.BaselineQ == nil || m.AggregateQ == nil || m.ObservedQ == nil {
				return 0, fmt.Errorf("missing required settlement inputs")
			}

			aggregate, ok := prev[KeyAggregateMetrics].(Result)
			if !ok || aggregate.Q == nil {
				return 0, fmt.Errorf("invalid aggregate metrics data")
			}
			baseline, ok := prev[KeyBaselineMetrics].(Result)
			if !ok || baseline.Q == nil {
				return 0, fmt.Errorf("invalid baseline metrics data")
			}
			scenarioA, ok := prev[KeyScenarioAInputs].(Result)
			if !ok || scenarioA.Q == nil {
				return 0, fmt.Errorf("invalid scenario A quantity")
			}
			observed, ok := prev[KeyObservedMetrics].(Result)
			if !ok || observed.Q == nil {
				return 0, fmt.Errorf("invalid observed quantity")
			}
			scenarioAPrice, ok := prev[KeyScenarioAInputs].(Result)
			if !ok || scenarioAPrice.P == nil {
				return 0, fmt.Errorf("invalid scenario A price")
			}
			scenarioBPrice, ok := prev[KeyScenarioBInputs].(Result)
			if !ok || scenarioBPrice.P == nil {
				return 0, fmt.Errorf("invalid scenario B price")
			}

			var result float64
			if float64(*scenarioAPrice.P) < float64(*scenarioBPrice.P) {
				diffQ := utils.DecimalSubtract(float64(*aggregate.Q), float64(*baseline.Q))
				diffQ = utils.DecimalSubtract(diffQ, float64(*scenarioA.Q))
				diffP := utils.DecimalSubtract(float64(*scenarioAPrice.P), float64(*scenarioBPrice.P))
				result = utils.DecimalMul(diffQ, diffP)
			} else {
				sumQ := utils.DecimalAdd(float64(*baseline.Q), float64(*scenarioA.Q))
				sumQ = utils.DecimalSubtract(sumQ, float64(*observed.Q))
				diffP := utils.DecimalSubtract(float64(*scenarioBPrice.P), float64(*scenarioAPrice.P))
				result = utils.DecimalMul(sumQ, diffP)
			}

			return result, nil
		},
	})

	// 场景收益：评估观测交付与场景假设差异带来的收益。
	RegisterFormula(FormulaNode{
		name: KeyScenarioMargin,
		deps: []string{
			KeyAggregateMetrics,
			KeyBaselineMetrics,
			KeyScenarioAInputs,
			KeyScenarioBInputs,
			KeyObservedMetrics,
		},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			if m.ScenarioAP == nil || m.ScenarioBP == nil || m.ScenarioAQ == nil || m.BaselineQ == nil || m.AggregateQ == nil || m.ObservedQ == nil {
				return 0, fmt.Errorf("missing required scenario margin inputs")
			}

			aggregate, ok := prev[KeyAggregateMetrics].(Result)
			if !ok || aggregate.Q == nil {
				return 0, fmt.Errorf("invalid aggregate metrics data")
			}
			baseline, ok := prev[KeyBaselineMetrics].(Result)
			if !ok || baseline.Q == nil {
				return 0, fmt.Errorf("invalid baseline metrics data")
			}
			scenarioA, ok := prev[KeyScenarioAInputs].(Result)
			if !ok || scenarioA.Q == nil {
				return 0, fmt.Errorf("invalid scenario A quantity")
			}
			observed, ok := prev[KeyObservedMetrics].(Result)
			if !ok || observed.Q == nil {
				return 0, fmt.Errorf("invalid observed quantity")
			}
			scenarioAPrice, ok := prev[KeyScenarioAInputs].(Result)
			if !ok || scenarioAPrice.P == nil {
				return 0, fmt.Errorf("invalid scenario A price")
			}
			scenarioBPrice, ok := prev[KeyScenarioBInputs].(Result)
			if !ok || scenarioBPrice.P == nil {
				return 0, fmt.Errorf("invalid scenario B price")
			}

			var result float64
			if float64(*scenarioAPrice.P) < float64(*scenarioBPrice.P) {
				observedAdjusted := utils.DecimalMul(float64(*observed.Q), 1.2)
				sumQ := utils.DecimalAdd(float64(*scenarioA.Q), float64(*baseline.Q))
				sumQ = utils.DecimalSubtract(sumQ, observedAdjusted)
				diffP := utils.DecimalSubtract(float64(*scenarioBPrice.P), float64(*scenarioAPrice.P))
				result = utils.DecimalMul(sumQ, diffP)
			} else {
				aggregateAdjusted := utils.DecimalMul(float64(*aggregate.Q), 0.8)
				diffQ := utils.DecimalSubtract(aggregateAdjusted, float64(*baseline.Q))
				diffQ = utils.DecimalSubtract(diffQ, float64(*scenarioA.Q))
				diffP := utils.DecimalSubtract(float64(*scenarioAPrice.P), float64(*scenarioBPrice.P))
				result = utils.DecimalMul(diffQ, diffP)
			}

			return result, nil
		},
	})

	// 总成本 = 基础成本 + 结算影响。
	RegisterFormula(FormulaNode{
		name: KeyTotalCost,
		deps: []string{KeyBaseCost, KeySettlementImpact},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			baseCost, ok := prev[KeyBaseCost].(float64)
			if !ok {
				return 0, fmt.Errorf("base cost is unavailable")
			}
			settlement, ok := prev[KeySettlementImpact].(float64)
			if !ok {
				return 0, fmt.Errorf("settlement impact is unavailable")
			}

			return utils.DecimalAdd(baseCost, settlement), nil
		},
	})

	// 净收益 = 结算影响 - 场景收益。
	RegisterFormula(FormulaNode{
		name: KeyNetMargin,
		deps: []string{KeySettlementImpact, KeyScenarioMargin},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			settlement, ok := prev[KeySettlementImpact].(float64)
			if !ok {
				return 0, fmt.Errorf("settlement impact is unavailable")
			}
			margin, ok := prev[KeyScenarioMargin].(float64)
			if !ok {
				return 0, fmt.Errorf("scenario margin is unavailable")
			}

			return utils.DecimalSubtract(settlement, margin), nil
		},
	})

	// 单位收益 = 净收益 / 汇总量。
	RegisterFormula(FormulaNode{
		name: KeyUnitYield,
		deps: []string{KeyNetMargin, KeyAggregateMetrics},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			if m.AggregateQ == nil {
				return 0, fmt.Errorf("context.AggregateQ is nil")
			}
			aggregate, ok := prev[KeyAggregateMetrics].(Result)
			if !ok || aggregate.Q == nil {
				return 0, fmt.Errorf("invalid aggregate metrics data")
			}
			netMargin, ok := prev[KeyNetMargin].(float64)
			if !ok {
				return 0, fmt.Errorf("net margin is unavailable")
			}

			if float64(*aggregate.Q) == 0 {
				return 0, nil
			}

			return utils.DecimalDivide(netMargin, float64(*aggregate.Q), 4), nil
		},
	})
}
