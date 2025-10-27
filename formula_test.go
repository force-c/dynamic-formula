package dynamicformula

import (
	"fmt"
	"testing"

	"github.com/force-c/dynamic-formula/utils"
	"github.com/shopspring/decimal"
)

func TestCalc_BaseCost(t *testing.T) {
	template := NewCalcTemplate(FormulaNode{
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
			return float64(*prev[KeyBaselineMetrics].(Result).V) +
				float64(*prev[KeyScenarioAInputs].(Result).V) +
				float64(*prev[KeyScenarioBInputs].(Result).V), nil
		},
	})

	input := ContextInput{
		BaselineV:  NewOptionalFloat(10),
		ScenarioAV: NewOptionalFloat(5),
		ScenarioBV: NewOptionalFloat(2),
	}

	data, err := input.Calc(template, true)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		t.Logf("key %s value %v", k, v)
	}
}

func TestCalc_TotalCost(t *testing.T) {
	template := NewCalcTemplate(FormulaNode{
		name: KeyTotalCost,
		deps: []string{KeyBaseCost, KeySettlementImpact},
		formula: func(m ContextInput, prev map[string]interface{}) (float64, error) {
			return prev[KeyBaseCost].(float64) + prev[KeySettlementImpact].(float64), nil
		},
	})

	input := ContextInput{
		AggregateQ: NewOptionalFloat(30),
		BaselineQ:  NewOptionalFloat(8),
		ScenarioAQ: NewOptionalFloat(4),
		ObservedQ:  NewOptionalFloat(12),
		ScenarioAP: NewOptionalFloat(20),
		ScenarioBP: NewOptionalFloat(18),
		BaselineV:  NewOptionalFloat(9),
		ScenarioAV: NewOptionalFloat(3),
		ScenarioBV: NewOptionalFloat(2),
	}

	data, err := input.Calc(template, true)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		t.Logf("key %s value %v", k, v)
	}
}

func TestCalc_SettlementImpact(t *testing.T) {
	f, _ := formulaRegistry[KeySettlementImpact]
	template := NewCalcTemplate(f)

	t.Run("scenario A price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(24),
			BaselineQ:  NewOptionalFloat(6),
			ScenarioAQ: NewOptionalFloat(5),
			ObservedQ:  NewOptionalFloat(7),
			ScenarioAP: NewOptionalFloat(22),
			ScenarioBP: NewOptionalFloat(18),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("scenario B price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(24),
			BaselineQ:  NewOptionalFloat(6),
			ScenarioAQ: NewOptionalFloat(5),
			ObservedQ:  NewOptionalFloat(9),
			ScenarioAP: NewOptionalFloat(18),
			ScenarioBP: NewOptionalFloat(22),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

func TestCalc_ScenarioMargin(t *testing.T) {
	f, _ := formulaRegistry[KeyScenarioMargin]
	template := NewCalcTemplate(f)

	t.Run("scenario A price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(12),
			BaselineQ:  NewOptionalFloat(3),
			ScenarioAQ: NewOptionalFloat(4),
			ObservedQ:  NewOptionalFloat(6),
			ScenarioAP: NewOptionalFloat(25),
			ScenarioBP: NewOptionalFloat(21),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("scenario B price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(12),
			BaselineQ:  NewOptionalFloat(3),
			ScenarioAQ: NewOptionalFloat(4),
			ObservedQ:  NewOptionalFloat(2),
			ScenarioAP: NewOptionalFloat(18),
			ScenarioBP: NewOptionalFloat(22),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

func TestCalc_NetMargin(t *testing.T) {
	f, _ := formulaRegistry[KeyNetMargin]
	template := NewCalcTemplate(f)

	t.Run("scenario A price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(14),
			BaselineQ:  NewOptionalFloat(4),
			ScenarioAQ: NewOptionalFloat(3),
			ObservedQ:  NewOptionalFloat(6),
			ScenarioAP: NewOptionalFloat(25),
			ScenarioBP: NewOptionalFloat(21),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("scenario B price higher", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(14),
			BaselineQ:  NewOptionalFloat(4),
			ScenarioAQ: NewOptionalFloat(3),
			ObservedQ:  NewOptionalFloat(2),
			ScenarioAP: NewOptionalFloat(18),
			ScenarioBP: NewOptionalFloat(22),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

func TestCalc_UnitYield(t *testing.T) {
	f, _ := formulaRegistry[KeyUnitYield]
	template := NewCalcTemplate(f)

	t.Run("non-zero aggregate quantity", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(16),
			BaselineQ:  NewOptionalFloat(5),
			ScenarioAQ: NewOptionalFloat(4),
			ObservedQ:  NewOptionalFloat(2),
			ScenarioAP: NewOptionalFloat(18),
			ScenarioBP: NewOptionalFloat(21),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})

	t.Run("zero aggregate quantity", func(t *testing.T) {
		input := ContextInput{
			AggregateQ: NewOptionalFloat(0),
			BaselineQ:  NewOptionalFloat(5),
			ScenarioAQ: NewOptionalFloat(4),
			ObservedQ:  NewOptionalFloat(2),
			ScenarioAP: NewOptionalFloat(18),
			ScenarioBP: NewOptionalFloat(21),
			BaselineV:  NewOptionalFloat(6),
			ScenarioAV: NewOptionalFloat(3),
			ScenarioBV: NewOptionalFloat(4),
			ObservedV:  NewOptionalFloat(5),
		}

		data, err := input.Calc(template, true)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range data {
			t.Logf("key %s value %v", k, v)
		}
	})
}

func TestCalc_FullTemplate(t *testing.T) {
	dataSets := []struct {
		Period int
		Data   ContextInput
	}{
		{
			Period: 1,
			Data: ContextInput{
				Period:     1,
				ObservedQ:  NewOptionalFloat(0.8),
				ObservedP:  NewOptionalFloat(19.2),
				ObservedV:  NewOptionalFloat(15.36),
				AggregateQ: NewOptionalFloat(0.8),
				AggregateP: NewOptionalFloat(20.0),
				AggregateV: NewOptionalFloat(16.0),
				BaselineQ:  NewOptionalFloat(0.2),
				BaselineP:  NewOptionalFloat(21.5),
				BaselineV:  NewOptionalFloat(4.3),
				ScenarioAQ: NewOptionalFloat(0.25),
				ScenarioAP: NewOptionalFloat(18.5),
				ScenarioAV: NewOptionalFloat(4.625),
				ScenarioBQ: NewOptionalFloat(0.35),
				ScenarioBP: NewOptionalFloat(17.8),
				ScenarioBV: NewOptionalFloat(6.23),
				OverheadQ:  NewOptionalFloat(0),
				OverheadP:  NewOptionalFloat(0),
				OverheadV:  NewOptionalFloat(0.8),
			},
		},
		{
			Period: 2,
			Data: ContextInput{
				Period:     2,
				ObservedQ:  NewOptionalFloat(1.1),
				ObservedP:  NewOptionalFloat(20.5),
				ObservedV:  NewOptionalFloat(22.55),
				AggregateQ: NewOptionalFloat(1.1),
				AggregateP: NewOptionalFloat(19.8),
				AggregateV: NewOptionalFloat(21.78),
				BaselineQ:  NewOptionalFloat(0.3),
				BaselineP:  NewOptionalFloat(22),
				BaselineV:  NewOptionalFloat(6.6),
				ScenarioAQ: NewOptionalFloat(0.4),
				ScenarioAP: NewOptionalFloat(19),
				ScenarioAV: NewOptionalFloat(7.6),
				ScenarioBQ: NewOptionalFloat(0.45),
				ScenarioBP: NewOptionalFloat(18.7),
				ScenarioBV: NewOptionalFloat(8.415),
				OverheadQ:  NewOptionalFloat(0),
				OverheadP:  NewOptionalFloat(0),
				OverheadV:  NewOptionalFloat(1.1),
			},
		},
		{
			Period: 3,
			Data: ContextInput{
				Period:     3,
				ObservedQ:  NewOptionalFloat(0.6),
				ObservedP:  NewOptionalFloat(18.1),
				ObservedV:  NewOptionalFloat(10.86),
				AggregateQ: NewOptionalFloat(0.6),
				AggregateP: NewOptionalFloat(18.9),
				AggregateV: NewOptionalFloat(11.34),
				BaselineQ:  NewOptionalFloat(0.15),
				BaselineP:  NewOptionalFloat(21.8),
				BaselineV:  NewOptionalFloat(3.27),
				ScenarioAQ: NewOptionalFloat(0.22),
				ScenarioAP: NewOptionalFloat(17.4),
				ScenarioAV: NewOptionalFloat(3.828),
				ScenarioBQ: NewOptionalFloat(0.28),
				ScenarioBP: NewOptionalFloat(18.3),
				ScenarioBV: NewOptionalFloat(5.124),
				OverheadQ:  NewOptionalFloat(0),
				OverheadP:  NewOptionalFloat(0),
				OverheadV:  NewOptionalFloat(0.6),
			},
		},
	}

	template := NewFullCalcTemplate()

	for _, ds := range dataSets {
		data, err := ds.Data.Calc(template, false)
		if err != nil {
			t.Fatalf("period %d evaluation failed: %v", ds.Period, err)
		}

		keys := []string{
			KeyBaseCost,
			KeySettlementImpact,
			KeyScenarioMargin,
			KeyTotalCost,
			KeyNetMargin,
			KeyUnitYield,
		}
		for _, key := range keys {
			if v, ok := data[key]; ok {
				t.Logf("period %d %s: %g", ds.Period, key, v)
			} else {
				t.Logf("period %d %s: <not found>", ds.Period, key)
			}
		}
		t.Log()
	}
}

func TestDecimalHelpers(t *testing.T) {
	sum, _ := decimal.NewFromFloat(2.5).
		Add(decimal.NewFromFloat(3.75)).
		Add(decimal.NewFromFloat(1.125)).
		Float64()
	t.Log("decimal sum ", sum)

	sum2 := utils.DecimalAdd(2.5, 3.75, 1.125)
	t.Log("helper sum ", sum2)
}
