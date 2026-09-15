package ollama

import (
	"reflect"
	"testing"
)

func TestSystemModelsPathsForGOOS(t *testing.T) {
	cases := []struct {
		goos string
		want []string
	}{
		{"linux", []string{"/usr/share/ollama/.ollama/models"}},
		{"darwin", nil},
		{"windows", nil},
		{"freebsd", nil},
		{"", nil},
	}
	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			got := systemModelsPathsForGOOS(tc.goos)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("systemModelsPathsForGOOS(%q) = %v, want %v", tc.goos, got, tc.want)
			}
		})
	}
}
