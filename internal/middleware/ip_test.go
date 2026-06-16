package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{name: "forwarded_for_first_ip", remoteAddr: "10.0.0.1:1234", xff: "1.2.3.4, 5.6.7.8", want: "1.2.3.4"},
		{name: "real_ip", remoteAddr: "10.0.0.1:1234", xri: "9.8.7.6", want: "9.8.7.6"},
		{name: "remote_addr_host", remoteAddr: "10.0.0.1:1234", want: "10.0.0.1"},
		{name: "raw_remote_addr", remoteAddr: "unix_socket", want: "unix_socket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			if got := getClientIP(req); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
