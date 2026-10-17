package update

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want Version
	}{
		{"v0.1.0", Version{0, 1, 0, "", ""}},
		{"v1.2.3", Version{1, 2, 3, "", ""}},
		{"v2.0.0-beta.1", Version{2, 0, 0, "beta.1", ""}},
		{"v1.0.0+build.123", Version{1, 0, 0, "", "build.123"}},
		{"1.0.0", Version{1, 0, 0, "", ""}},
	}

	for _, c := range cases {
		got, err := ParseVersion(c.in)
		if err != nil {
			t.Errorf("ParseVersion(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseVersion(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "v0.1.0", 0},
		{"v0.2.0", "v0.1.0", 1},
		{"v0.1.0", "v0.2.0", -1},
		{"v1.0.0", "v0.9.9", 1},
		{"v0.1.1", "v0.1.0", 1},
		{"v1.0.0", "v1.0.0-beta", 1},
		{"v1.0.0-beta", "v1.0.0", -1},
	}

	for _, c := range cases {
		a, _ := ParseVersion(c.a)
		b, _ := ParseVersion(c.b)
		got := a.Compare(b)
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version{0, 2, 0, "", ""}
	if s := v.String(); s != "v0.2.0" {
		t.Errorf("String() = %q, want v0.2.0", s)
	}
}
