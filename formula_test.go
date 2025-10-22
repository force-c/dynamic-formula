package dynamicformula

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// TestTTLCache_SetGetDelete 测试 TTLCache 的基本功能
func TestTTLCache_SetGetDelete(t *testing.T) {
	cache := NewTTLCache()
	defer cache.Stop()

	key := "test_key"
	value := "test_value"
	ttl := 1 * time.Second

	err := cache.Set(key, value, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found := cache.Get(key)
	if !found {
		t.Fatal("Get: value not found immediately after Set")
	}
	if got != value {
		t.Fatalf("Get: expected %v, got %v", value, got)
	}

	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, found = cache.Get(key)
	if found {
		t.Fatal("Delete: key should be removed")
	}
}

// TestTTLCache_Expiration 测试 TTL 过期逻辑
func TestTTLCache_Expiration(t *testing.T) {
	cache := NewTTLCache()
	defer cache.Stop()

	key := "expire_key"
	value := 123
	ttl := 500 * time.Millisecond

	err := cache.Set(key, value, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found := cache.Get(key)
	if !found || got != value {
		t.Fatalf("Get: expected %v, got %v, found=%v", value, got, found)
	}

	time.Sleep(600 * time.Millisecond)

	_, found = cache.Get(key)
	if found {
		t.Fatal("Get: key should have expired")
	}
}

// TestTTLCache_ComplexValue 测试复杂类型存储
func TestTTLCache_ComplexValue(t *testing.T) {
	cache := NewTTLCache()
	defer cache.Stop()

	type testStruct struct {
		Name string
		Age  int
	}
	key := "struct_key"
	value := testStruct{Name: "Alice", Age: 30}
	ttl := 1 * time.Second

	err := cache.Set(key, value, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found := cache.Get(key)
	if !found {
		t.Fatal("Get: value not found")
	}
	gotStruct := got.(testStruct)
	if !reflect.DeepEqual(gotStruct, value) {
		t.Fatalf("Get: expected %+v, got %+v", value, gotStruct)
	}
}

// TestTTLCache_NodeSlice 测试缓存 Node 切片
func TestTTLCache_NodeSlice(t *testing.T) {
	cache := NewTTLCache()
	defer cache.Stop()

	node := FormulaNode{
		name:    KeyOriginalF,
		deps:    []string{KeyActualF},
		formula: func(m MomentData, prev map[string]Result) float64 { return 0 },
	}
	nodes := []Node{&node}

	key := "node_slice_key"
	ttl := 1 * time.Second

	err := cache.Set(key, nodes, ttl)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found := cache.Get(key)
	if !found {
		t.Fatal("Get: node slice not found")
	}
	gotNodes := got.([]Node)
	if len(gotNodes) != len(nodes) {
		t.Fatalf("Expected %d nodes, got %d", len(nodes), len(gotNodes))
	}
	if gotNodes[0].Name() != node.Name() {
		t.Fatalf("Expected node name %s, got %s", node.Name(), gotNodes[0].Name())
	}
}

// TestCalcTemplate_TopoSort 测试拓扑排序
func TestCalcTemplate_TopoSort(t *testing.T) {
	node1 := FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyActualF},
		formula: func(m MomentData, prev map[string]Result) float64 {
			return prev[KeyActualF].Value
		},
	}
	node2 := FormulaNode{
		name: KeyDeviationSettle,
		deps: []string{KeyOriginalF, KeyDADevF},
		formula: func(m MomentData, prev map[string]Result) float64 {
			return prev[KeyOriginalF].Value + prev[KeyDADevF].Value
		},
	}

	template := NewCalcTemplate(&node1, &node2)
	ordered, err := template.GetOrderedNodes()
	if err != nil {
		t.Fatalf("topoSortTemplate failed: %v", err)
	}

	// 检查排序顺序
	baseNodes := []string{KeyActualF, KeyDADevF, KeyLongTermF, KeyRTDevF, KeyTotalF, KeyTransferF}
	expectedOrder := append(baseNodes, KeyOriginalF, KeyDeviationSettle)
	if len(ordered) != len(expectedOrder) {
		t.Fatalf("Expected %d nodes, got %d", len(expectedOrder), len(ordered))
	}
	for i, node := range ordered {
		if node.Name() != expectedOrder[i] {
			t.Errorf("Expected node %s at position %d, got %s", expectedOrder[i], i, node.Name())
		}
	}
}

// TestCalcTemplate_Cache 测试缓存命中
func TestCalcTemplate_Cache(t *testing.T) {
	node := FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyActualF},
		formula: func(m MomentData, prev map[string]Result) float64 {
			return prev[KeyActualF].Value
		},
	}
	template := NewCalcTemplate(&node)

	ordered1, err := template.GetOrderedNodes()
	if err != nil {
		t.Fatalf("GetOrderedNodes failed: %v", err)
	}

	ordered2, err := template.GetOrderedNodes()
	if err != nil {
		t.Fatalf("GetOrderedNodes failed: %v", err)
	}

	if !reflect.DeepEqual(ordered1, ordered2) {
		t.Fatal("Cached order differs from computed order")
	}
}

// TestMomentData_Calc 测试单次计算
func TestMomentData_Calc(t *testing.T) {
	node := FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyActualF},
		formula: func(m MomentData, prev map[string]Result) float64 {
			return prev[KeyActualF].Value * 2
		},
	}
	template := NewCalcTemplate(&node)

	data := MomentData{
		Period:  1,
		ActualF: 100.0,
	}

	results, err := data.Calc(template)
	if err != nil {
		t.Fatalf("Calc failed: %v", err)
	}

	if val, ok := results[KeyActualF]; !ok || val.Value != 100.0 {
		t.Errorf("Expected ActualF=100.0, got %v", val.Value)
	}
	if val, ok := results[KeyOriginalF]; !ok || val.Value != 200.0 {
		t.Errorf("Expected OriginalF=200.0, got %v", val.Value)
	}
}

// TestCalcBatch 测试批量并发计算
func TestCalcBatch(t *testing.T) {
	node := FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyActualF},
		formula: func(m MomentData, prev map[string]Result) float64 {
			return prev[KeyActualF].Value * 2
		},
	}
	template := NewCalcTemplate(&node)

	dataList := []MomentData{
		{Period: 1, ActualF: 100.0},
		{Period: 2, ActualF: 200.0},
	}
	templates := []*CalcTemplate{template, template}

	results, errors := CalcBatch(dataList, templates)
	if len(errors) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(errors))
	}
	for i, err := range errors {
		if err != nil {
			t.Errorf("CalcBatch error at index %d: %v", i, err)
		}
	}

	if val := results[0][KeyOriginalF].Value; val != 200.0 {
		t.Errorf("Expected OriginalF=200.0, got %v", val)
	}
	if val := results[1][KeyOriginalF].Value; val != 400.0 {
		t.Errorf("Expected OriginalF=400.0, got %v", val)
	}
}

// TestTTLCache_ConcurrentAccess 测试并发读写
func TestTTLCache_ConcurrentAccess(t *testing.T) {
	cache := NewTTLCache()
	defer cache.Stop()

	var wg sync.WaitGroup
	key := "concurrent_key"
	value := "concurrent_value"
	ttl := 5 * time.Second

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cache.Set(key, value, ttl)
			if err != nil {
				t.Errorf("Concurrent Set failed: %v", err)
			}
		}()
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, found := cache.Get(key)
			if !found {
				t.Error("Concurrent Get: value not found")
			}
		}()
	}
	wg.Wait()
}
