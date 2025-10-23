package dynamicformula

import (
	"fmt"
	"testing"

	"github.com/force-c/dynamic-formula/utils"
	"github.com/shopspring/decimal"
)

// 原电费
func TestCalc_OriginalFe(t *testing.T) {
	template := NewCalcTemplate(FormulaNode{
		name: KeyOriginalF,
		deps: []string{KeyLongTermF, KeyDADevF, KeyRTDevF},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			// 检查依赖字段
			if m.LongTermF == nil {
				return 0, fmt.Errorf("field LongTermF is not set")
			}
			if m.DADevF == nil {
				return 0, fmt.Errorf("field DADevF is not set")
			}
			if m.RTDevF == nil {
				return 0, fmt.Errorf("field RTDevF is not set")
			}
			result := float64(*prev[KeyLongTermF].(Result).F) +
				float64(*prev[KeyDADevF].(Result).F) +
				float64(*prev[KeyRTDevF].(Result).F)
			return result, nil
		},
	})

	md := MomentData{
		LongTermF: NewOptionalFloat(5),
		DADevF:    NewOptionalFloat(5),
		RTDevF:    NewOptionalFloat(5),
	}

	data, err := md.Calc(template, true)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		t.Logf("key %s value %v", k, v)
	}
}

// 总电费
func TestCalc_KeyTotalFee(t *testing.T) {
	template := NewCalcTemplate(FormulaNode{
		name: KeyTotalFee,
		deps: []string{KeyOriginalF, KeyDeviationSettle},
		formula: func(m MomentData, prev map[string]interface{}) (float64, error) {
			return prev[KeyOriginalF].(float64) + prev[KeyDeviationSettle].(float64), nil
		},
	})

	md := MomentData{
		TotalQ:    NewOptionalFloat(30),
		LongTermQ: NewOptionalFloat(5),
		DADevQ:    NewOptionalFloat(5),
		DADevP:    NewOptionalFloat(38),
		RTDevP:    NewOptionalFloat(35),
		LongTermF: NewOptionalFloat(5),
		DADevF:    NewOptionalFloat(5),
		RTDevF:    NewOptionalFloat(5),
	}

	data, err := md.Calc(template, true)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		t.Logf("key %s value %v", k, v)
	}
}

// 偏差结算费用
func TestCalc_DeviationSettle(t *testing.T) {
	f, _ := formulaRegistry[KeyDeviationSettle]
	template := NewCalcTemplate(f)

	t.Run("偏差结算费用 日前大于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(30),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(10),
			DADevP:    NewOptionalFloat(38),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("偏差结算费用 日前小于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(30),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(3),
			DADevP:    NewOptionalFloat(30),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

// 偏差收益
func TestCalc_KeyDeviationProfit(t *testing.T) {
	f, _ := formulaRegistry[KeyDeviationProfit]
	template := NewCalcTemplate(f)

	t.Run("偏差结算费用 日前大于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(5),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(10),
			DADevP:    NewOptionalFloat(38),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("偏差结算费用 日前小于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(30),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(3),
			DADevP:    NewOptionalFloat(30),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

// 最终收益
func TestCalc_FinalProfit(t *testing.T) {
	f, _ := formulaRegistry[KeyFinalProfit]
	template := NewCalcTemplate(f)

	t.Run("最终收益 日前大于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(5),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(10),
			DADevP:    NewOptionalFloat(38),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("最终收益 日前小于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(30),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(3),
			DADevP:    NewOptionalFloat(30),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

// 套利
func TestCalc_Arbitrage(t *testing.T) {
	f, _ := formulaRegistry[KeyArbitrage]
	template := NewCalcTemplate(f)

	t.Run("套利 日前大于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(5),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(10),
			DADevP:    NewOptionalFloat(38),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("套利 日前小于实时", func(t *testing.T) {
		md := MomentData{
			TotalQ:    NewOptionalFloat(30),
			LongTermQ: NewOptionalFloat(5),
			DADevQ:    NewOptionalFloat(5),
			ActualQ:   NewOptionalFloat(3),
			DADevP:    NewOptionalFloat(30),
			RTDevP:    NewOptionalFloat(35),
			LongTermF: NewOptionalFloat(5),
			DADevF:    NewOptionalFloat(5),
			RTDevF:    NewOptionalFloat(5),
			ActualF:   NewOptionalFloat(5),
		}

		data, err := md.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

// 单元测试
func Test_excel(t *testing.T) {
	// 定义三个时刻的测试数据
	dataSets := []struct {
		Period int
		Data   MomentData
	}{
		{
			Period: 3,
			Data: MomentData{
				Period:    3,
				ActualQ:   NewOptionalFloat(0.06),
				ActualP:   nil,
				ActualF:   nil,
				TotalQ:    NewOptionalFloat(0.06),
				TotalP:    NewOptionalFloat(308.04681),
				TotalF:    NewOptionalFloat(18.4828086),
				LongTermQ: NewOptionalFloat(0.006),
				LongTermP: NewOptionalFloat(372),
				LongTermF: NewOptionalFloat(2.232),
				DADevQ:    NewOptionalFloat(0.0241),
				DADevP:    NewOptionalFloat(325.1897),
				DADevF:    NewOptionalFloat(7.83707177),
				RTDevQ:    NewOptionalFloat(0.0299),
				RTDevP:    NewOptionalFloat(216.0701),
				RTDevF:    NewOptionalFloat(6.46049599),
				TransferQ: NewOptionalFloat(0),
				TransferP: NewOptionalFloat(0),
				TransferF: NewOptionalFloat(1.95324084),
			},
		},
		{
			Period: 4,
			Data: MomentData{
				Period:    4,
				ActualQ:   NewOptionalFloat(0.06),
				ActualP:   nil,
				ActualF:   nil,
				TotalQ:    NewOptionalFloat(0.06),
				TotalP:    NewOptionalFloat(310.38966),
				TotalF:    NewOptionalFloat(18.6233796),
				LongTermQ: NewOptionalFloat(0.006),
				LongTermP: NewOptionalFloat(372),
				LongTermF: NewOptionalFloat(2.232),
				DADevQ:    NewOptionalFloat(0.0241),
				DADevP:    NewOptionalFloat(311.2202),
				DADevF:    NewOptionalFloat(7.50040682),
				RTDevQ:    NewOptionalFloat(0.0299),
				RTDevP:    NewOptionalFloat(276.6776),
				RTDevF:    NewOptionalFloat(8.27266024),
				TransferQ: NewOptionalFloat(0),
				TransferP: NewOptionalFloat(0),
				TransferF: NewOptionalFloat(0.61831254),
			},
		},
		{
			Period: 5,
			Data: MomentData{
				Period:    5,
				ActualQ:   NewOptionalFloat(0.47),
				ActualP:   nil,
				ActualF:   nil,
				TotalQ:    NewOptionalFloat(0.47),
				TotalP:    NewOptionalFloat(313.3031),
				TotalF:    NewOptionalFloat(18.798186),
				LongTermQ: NewOptionalFloat(0.006),
				LongTermP: NewOptionalFloat(372),
				LongTermF: NewOptionalFloat(2.232),
				DADevQ:    NewOptionalFloat(0.37),
				DADevP:    NewOptionalFloat(315.7922),
				DADevF:    NewOptionalFloat(116.843114),
				RTDevQ:    NewOptionalFloat(0.09),
				RTDevP:    NewOptionalFloat(320.7428),
				RTDevF:    NewOptionalFloat(28.866852),
				TransferQ: NewOptionalFloat(0),
				TransferP: NewOptionalFloat(0),
				TransferF: NewOptionalFloat(0.16198426),
			},
		},
	}

	// 创建计算模板
	template := NewFullCalcTemplate()

	// 遍历每个时刻的数据进行计算
	for _, ds := range dataSets {
		data, err := ds.Data.Calc(template, false)
		if err != nil {
			t.Fatalf("时刻 %d 计算失败: %v", ds.Period, err)
		}

		// 按固定顺序打印结果，确保日志清晰
		keys := []string{KeyOriginalF, KeyDeviationSettle, KeyDeviationProfit, KeyTotalFee, KeyFinalProfit, KeyArbitrage}
		for _, key := range keys {
			if v, ok := data[key]; ok {
				// 格式化输出，保留 6 位小数以保持一致性
				t.Logf("时刻 %d %s: %g", ds.Period, key, v)
			} else {
				t.Logf("时刻 %d %s: <not found>", ds.Period, key)
			}
		}
		t.Log()
	}
}

func Test_dec(t *testing.T) {
	f, _ := decimal.NewFromFloat(2.232).
		Add(decimal.NewFromFloat(7.83707177)).
		Add(decimal.NewFromFloat(6.46049599)).
		Float64()
	t.Log("结果1 ", f)

	f2 := utils.DecimalAdd(2.232, 7.83707177, 6.46049599)
	//f2, _ := decimal.NewFromFloat(2.232).
	//	Add(decimal.NewFromFloat(7.83707177)).
	//	Add(decimal.NewFromFloat(6.46049599)).
	//	Float64()
	t.Log("结果2 ", f2)
}
