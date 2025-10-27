# Dynamic Formula / 动态公式框架

[English](#english) | [中文](#中文)

---

## 中文

一个通用的Go语言计算图框架，支持依赖管理。通过定义输入节点和公式节点来创建复杂的计算流水线，具备自动依赖解析和执行排序功能。

### 功能特性

- **通用框架**: 构建任意类型的计算流水线，不局限于特定业务领域
- **依赖管理**: 自动依赖解析和拓扑排序
- **节点注册**: 简单的API注册输入适配器和公式节点
- **模板系统**: 创建可复用的计算模板
- **类型安全**: 强类型系统和自定义结果结构
- **缓存机制**: 内置TTL缓存提升性能
- **并行执行**: 独立计算的高效执行

### 核心概念

#### 节点
- **输入节点**: 使用 `InputAdapter` 函数将上下文数据转换为标准化结果
- **公式节点**: 执行依赖其他节点的自定义计算

#### 模板
- **CalcTemplate**: 节点集合，包含已解析的依赖关系
- **执行顺序**: 根据节点依赖关系自动确定

### 基本用法

#### 1. 注册输入节点

```go
// 注册自定义输入适配器
RegisterInputNode("user_metrics", func(ctx ContextInput) (q, p, v *OptionalFloat) {
    return ctx.ObservedQ, ctx.ObservedP, ctx.ObservedV
})

// 替代语法
RegisterInputAdapter("custom_input", func(ctx ContextInput) (q, p, v *OptionalFloat) {
    // 自定义输入逻辑
    return NewOptionalFloat(1.0), NewOptionalFloat(2.0), NewOptionalFloat(3.0)
})
```

#### 2. 注册公式节点

```go
// 注册自定义计算公式
RegisterFormula(FormulaNode{
    name: "my_calculation",
    deps: []string{"user_metrics", "other_input"},
    formula: func(ctx ContextInput, prev map[string]interface{}) (float64, error) {
        // 访问依赖结果
        userMetrics := prev["user_metrics"].(Result)
        otherInput := prev["other_input"].(Result)
        
        // 计算逻辑
        if userMetrics.V != nil && otherInput.V != nil {
            return float64(*userMetrics.V) + float64(*otherInput.V), nil
        }
        return 0, fmt.Errorf("缺少必需的值")
    },
})
```

#### 3. 创建和执行模板

```go
// 创建计算模板
template := NewCalcTemplate(formulaRegistry["my_calculation"])

// 准备输入上下文
input := ContextInput{
    ObservedQ: NewOptionalFloat(10.0),
    ObservedP: NewOptionalFloat(5.0),
    ObservedV: NewOptionalFloat(50.0),
}

// 执行计算
results, err := input.Calc(template, false)
if err != nil {
    log.Fatal(err)
}

// 访问结果
fmt.Printf("my_calculation 结果: %v\n", results["my_calculation"])
```

### 扩展性

框架设计为可扩展到任何领域：

1. **定义上下文**: 扩展 `ContextInput` 或创建自定义输入结构
2. **创建输入适配器**: 将数据转换为框架的 `Result` 格式
3. **构建公式节点**: 将业务逻辑实现为公式函数
4. **注册组件**: 使用 `RegisterInputNode()` 和 `RegisterFormula()` 使组件可用
5. **执行**: 创建模板并运行计算

### 内置示例

框架包含金融计算示例，展示了：
- 成本计算 (`base_cost`, `total_cost`)
- 影响分析 (`settlement_impact`, `scenario_margin`)
- 收益计算 (`net_margin`, `unit_yield`)

这些作为参考实现，可以为你的特定用例替换或扩展。

### 开发指南

为你的领域适配此框架：

1. 查看 `formula.go:347-578` 中的现有公式实现
2. 研究 `formula_test.go` 中的测试用例了解使用模式
3. 用你的领域特定逻辑替换或扩展内置公式
4. 修改 `ContextInput` 结构以匹配你的数据需求

### 测试

```bash
go test -v
```

### 许可证

开源项目 - 可自由适配你的特定计算需求。

---

## English

A generic Go framework for building computation graphs with dependencies. Define input nodes and formula nodes to create complex calculation pipelines with automatic dependency resolution and execution ordering.

### Features

- **Generic Framework**: Build any type of calculation pipeline, not limited to specific business domains
- **Dependency Management**: Automatic dependency resolution and topological sorting
- **Node Registration**: Simple APIs to register input adapters and formula nodes
- **Template System**: Create reusable calculation templates
- **Type Safety**: Strong typing with custom result structures
- **Caching**: Built-in TTL cache for performance optimization
- **Parallel Execution**: Efficient execution of independent calculations

### Core Concepts

#### Nodes
- **Input Nodes**: Transform context data into standardized results using `InputAdapter` functions
- **Formula Nodes**: Execute custom calculations with dependencies on other nodes

#### Templates
- **CalcTemplate**: Collections of nodes with resolved dependencies
- **Execution Order**: Automatically determined based on node dependencies

### Basic Usage

#### 1. Register Input Nodes

```go
// Register custom input adapters
RegisterInputNode("user_metrics", func(ctx ContextInput) (q, p, v *OptionalFloat) {
    return ctx.ObservedQ, ctx.ObservedP, ctx.ObservedV
})

// Alternative syntax
RegisterInputAdapter("custom_input", func(ctx ContextInput) (q, p, v *OptionalFloat) {
    // Your custom input logic
    return NewOptionalFloat(1.0), NewOptionalFloat(2.0), NewOptionalFloat(3.0)
})
```

#### 2. Register Formula Nodes

```go
// Register custom calculation formulas
RegisterFormula(FormulaNode{
    name: "my_calculation",
    deps: []string{"user_metrics", "other_input"},
    formula: func(ctx ContextInput, prev map[string]interface{}) (float64, error) {
        // Access dependency results
        userMetrics := prev["user_metrics"].(Result)
        otherInput := prev["other_input"].(Result)
        
        // Your calculation logic
        if userMetrics.V != nil && otherInput.V != nil {
            return float64(*userMetrics.V) + float64(*otherInput.V), nil
        }
        return 0, fmt.Errorf("missing required values")
    },
})
```

#### 3. Create and Execute Templates

```go
// Create calculation template
template := NewCalcTemplate(formulaRegistry["my_calculation"])

// Prepare input context
input := ContextInput{
    ObservedQ: NewOptionalFloat(10.0),
    ObservedP: NewOptionalFloat(5.0),
    ObservedV: NewOptionalFloat(50.0),
}

// Execute calculations
results, err := input.Calc(template, false)
if err != nil {
    log.Fatal(err)
}

// Access results
fmt.Printf("my_calculation result: %v\n", results["my_calculation"])
```

### Extensibility

The framework is designed to be extended for any domain:

1. **Define Your Context**: Extend `ContextInput` or create custom input structures
2. **Create Input Adapters**: Transform your data into the framework's `Result` format
3. **Build Formula Nodes**: Implement your business logic as formula functions
4. **Register Components**: Use `RegisterInputNode()` and `RegisterFormula()` to make components available
5. **Execute**: Create templates and run calculations

### Built-in Example

The framework includes a financial calculation example demonstrating:
- Cost calculations (`base_cost`, `total_cost`)
- Impact analysis (`settlement_impact`, `scenario_margin`)
- Yield calculations (`net_margin`, `unit_yield`)

These serve as reference implementations that can be replaced or extended for your specific use case.

### Development

To adapt this framework for your domain:

1. Review the existing formula implementations in `formula.go:347-578`
2. Study the test cases in `formula_test.go` for usage patterns
3. Replace or extend the built-in formulas with your domain-specific logic
4. Modify `ContextInput` structure to match your data requirements

### Testing

```bash
go test -v
```

### License

Open source - feel free to adapt for your specific calculation requirements.