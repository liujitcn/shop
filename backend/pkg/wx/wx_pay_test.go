package wx

import (
	"errors"
	"testing"

	wxPayCore "github.com/wechatpay-apiv3/wechatpay-go/core"
)

func TestIsPayOrderNotExist(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "order not exist", err: &wxPayCore.APIError{Code: "ORDER_NOT_EXIST"}, want: true},
		{name: "other api error", err: &wxPayCore.APIError{Code: "SYSTEM_ERROR"}},
		{name: "generic error", err: errors.New("network error")},
		{name: "nil error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPayOrderNotExist(test.err); got != test.want {
				t.Fatalf("IsPayOrderNotExist() = %t, want %t", got, test.want)
			}
		})
	}
}
