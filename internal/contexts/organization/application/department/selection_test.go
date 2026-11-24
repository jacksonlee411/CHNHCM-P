package department

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentCSVPropagatesOutputFailureAndCancellation(t *testing.T) {
	timeline := &d.Timeline{ID: foundation.NewID(), Code: "=FORMULA"}
	record := &d.Record{Data: d.Data{Name: "名称,含逗号"}}
	member := listMember{t: timeline, r: record, fields: readCode | readName}
	failure := errors.New("storage full")
	if err := departmentCSV(context.Background(), failingCSVWriter{failure}, []listMember{member}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := departmentCSV(context.Background(), &output, []listMember{member}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "'=FORMULA") || !strings.Contains(output.String(), `"名称,含逗号"`) {
		t.Fatal(output.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output.Reset()
	if err := departmentCSV(ctx, &output, []listMember{member}); !errors.Is(err, context.Canceled) || output.Len() != 0 {
		t.Fatal(err, output.Len())
	}
	if err := departmentCSV(context.Background(), io.Discard, []listMember{member}); err != nil {
		t.Fatal(err)
	}
}

type failingCSVWriter struct{ err error }

func (w failingCSVWriter) Write([]byte) (int, error) { return 0, w.err }
