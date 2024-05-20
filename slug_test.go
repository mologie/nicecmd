package nicecmd

import "testing"

func Test_slug(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{

		{"CamelCase", "camel-case"},
		{"CamelCamelCase", "camel-camel-case"},
		{"Camel2Camel2Case", "camel2-camel2-case"},
		{"PathToCSV", "path-to-csv"},
		{"CAPath", "ca-path"},
		{"EndsInUppeR", "ends-in-uppe-r"},
		{"eNdSiNLower", "e-nd-si-n-lower"},
		{"ALLUPPER", "allupper"},
		{"alllower", "alllower"},
		{"firstNotLower", "first-not-lower"},
		{"IP", "ip"},
		{"IPMask", "ip-mask"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := slug(tt.in, '-'); got != tt.want {
				t.Errorf("slug(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func Test_screamingSnake(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"CamelCase", "CAMEL_CASE"},
		{"CamelCamelCase", "CAMEL_CAMEL_CASE"},
		{"Camel2Camel2Case", "CAMEL2_CAMEL2_CASE"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := screamingSnake(tt.in); got != tt.want {
				t.Errorf("screamingSnake(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
