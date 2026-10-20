package manifest_test

import (
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

func TestParseStartsEachServiceAfterItsDependencies(t *testing.T) {
	tests := []struct {
		name  string
		specs []string
		want  []string
	}{
		{"independent services keep their order", []string{"b", "a", "c"}, []string{"b", "a", "c"}},
		{"a chain", []string{"a:b", "b:c", "c"}, []string{"c", "b", "a"}},
		{"a diamond", []string{"d:b,c", "b:a", "c:a", "a"}, []string{"a", "b", "c", "d"}},
		{"a repeated dependency", []string{"b:a,a", "a"}, []string{"a", "b"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			services, err := manifest.Parse([]byte(manifestOf(test.specs...)))
			if err != nil {
				t.Fatal(err)
			}

			if got := names(services); !slices.Equal(got, test.want) {
				t.Errorf("start order = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseNamesADependencyCycle(t *testing.T) {
	tests := []struct {
		name  string
		specs []string
		want  string
	}{
		{"two services", []string{"a:b", "b:a"}, "dependency cycle: a → b → a"},
		{"three services", []string{"a:b", "b:c", "c:a"}, "dependency cycle: a → b → c → a"},
		{"reached from outside", []string{"x:a", "a:b", "b:a"}, "dependency cycle: a → b → a"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			services, err := manifest.Parse([]byte(manifestOf(test.specs...)))
			if err == nil {
				t.Fatalf("got %+v, want the error %q", services, test.want)
			}

			if err.Error() != test.want {
				t.Errorf("got the error %q, want %q", err, test.want)
			}
		})
	}
}
