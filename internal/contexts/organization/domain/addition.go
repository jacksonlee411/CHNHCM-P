package domain

import "chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"

type AdditionWindow = dated.AdditionWindow[BUData]

func PrepareAddition(base []Record, anchor, enterpriseStart, today string) (AdditionWindow, error) {
	w, e := buKernel.PrepareAddition(base, anchor, enterpriseStart, today)
	return w, e
}
func ValidateAdditionPosition(base []Record, anchor, enterpriseStart, date string) error {
	return buKernel.ValidateAdditionPosition(base, anchor, enterpriseStart, date)
}
