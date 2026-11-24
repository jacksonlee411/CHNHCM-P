package department

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"testing"
)

func TestQueryPathFailurePreservesCauseAndPublicBoundary(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, errors.New("private missing path identity")} {
		got := unavailableQueryPath(cause)
		if !errors.Is(got, cause) {
			t.Fatal("path failure cause lost")
		}
		var public *foundation.Fault
		if !errors.As(got, &public) || public.Status != 503 || public.Code != "M02_DEPENDENCY_UNAVAILABLE" || public.Message != "部门路径暂时无法加载，请重试" {
			t.Fatal("unsafe or changed public fault", public)
		}
	}
}
