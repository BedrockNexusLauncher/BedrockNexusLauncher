package uwp

import "testing"

func TestErrorCodePassthrough(t *testing.T) {
    err := failure("ERR_UWP_DEV_MODE", nil)
    if ErrorCode(err) != "ERR_UWP_DEV_MODE" {
        t.Fatalf("expected ERR_UWP_DEV_MODE, got %q", ErrorCode(err))
    }
    if ErrorCode(nil) != "" {
        t.Fatal("expected empty code for nil")
    }
}
