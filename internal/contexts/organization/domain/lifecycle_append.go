package domain

func PrepareLifecycleAppend(base []Record, anchor, kind, enterpriseStart, today string) (AdditionWindow, error) {
	w, e := buKernel.PrepareLifecycleAppend(base, anchor, kind, enterpriseStart, today)
	return w, e
}
func ValidateLifecycleAppend(base []Record, anchor, kind, enterpriseStart, date string) error {
	return buKernel.ValidateLifecycleAppend(base, anchor, kind, enterpriseStart, date)
}
