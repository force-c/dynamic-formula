package utils

import "github.com/shopspring/decimal"

func DecimalAdd(values ...float64) float64 {
	var sum decimal.Decimal
	for _, value := range values {
		valueDecimal := decimal.NewFromFloat(value)
		sum = sum.Add(valueDecimal)
	}
	result, _ := sum.Float64()
	return result
}

func DecimalSubtract(value1 float64, value2 float64) float64 {
	value1Decimal := decimal.NewFromFloat(value1)
	value2Decimal := decimal.NewFromFloat(value2)
	result, _ := value1Decimal.Sub(value2Decimal).Float64()
	return result
}

func DecimalMul(value1 float64, value2 float64) float64 {
	if value2 == 0 {
		return 0
	}
	value1Decimal := decimal.NewFromFloat(value1)
	value2Decimal := decimal.NewFromFloat(value2)
	result, _ := value1Decimal.Mul(value2Decimal).Float64()
	return result
}

func DecimalDivide(value1 float64, value2 float64, reserve int) float64 {
	if value2 == 0 {
		return 0
	}
	value1Decimal := decimal.NewFromFloat(value1)
	value2Decimal := decimal.NewFromFloat(value2)
	result, _ := value1Decimal.Div(value2Decimal).Round(int32(reserve)).Float64()
	return result
}
