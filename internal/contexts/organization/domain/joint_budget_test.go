package domain

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestJointLongTimelineCancellationBudget(t *testing.T) {
	base := make([]Record, 120000)
	start, _ := time.Parse("2006-01-02", "2020-01-01")
	for i := range base {
		kind := "CHANGE"
		if i == 0 {
			kind = "CREATE"
		}
		end := ""
		if i+1 < len(base) {
			end = start.AddDate(0, 0, (i+1)*2).Format("2006-01-02")
		}
		base[i] = Record{ID: fmt.Sprint(i), Revision: "v1", Date: start.AddDate(0, 0, i*2).Format("2006-01-02"), End: end, Type: kind, State: "active", Data: BUData{Name: "甲"}}
	}
	items := []Item{{ID: "a", Action: "UPDATE", RecordID: "60000", ExpectedRevision: "v1", Changes: PatchFromFields(map[string]*string{"name": textValue("乙")})}, {ID: "b", Action: "UPDATE", RecordID: "90000", ExpectedRevision: "v1", Date: start.AddDate(0, 0, 180001).Format("2006-01-02")}}
	var maximum time.Duration
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, e := Calculate(ctx, base, "0", "2020-01-01", items); done <- e }()
		time.Sleep(time.Millisecond)
		at := time.Now()
		cancel()
		e := <-done
		elapsed := time.Since(at)
		if !errors.Is(e, context.Canceled) || elapsed > time.Second {
			t.Fatal(e, elapsed)
		}
		if elapsed > maximum {
			maximum = elapsed
		}
	}
	t.Logf("120000 records, two UPDATE items, five cancellations; maximum local stop=%s (<1s)", maximum)
}
