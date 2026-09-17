package ebay

import "strconv"

func lessMoney(a, b Money) bool {
	av, aerr := strconv.ParseFloat(a.Value, 64)
	bv, berr := strconv.ParseFloat(b.Value, 64)
	if aerr != nil || berr != nil {
		return false
	}
	return av < bv
}
