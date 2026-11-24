package domain

func CorrectionBounds(base []Record, target, enterpriseStart string) (string, string, error) {
	return buKernel.CorrectionBounds(base, target, enterpriseStart)
}
func ValidateCorrectionDate(base []Record, target, enterpriseStart, date string) error {
	return buKernel.ValidateCorrectionDate(base, target, enterpriseStart, date)
}
