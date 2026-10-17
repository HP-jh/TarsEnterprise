package update

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	cases := []struct {
		input   string
		want    Semver
		wantErr bool
	}{
		{"v1.2.3", Semver{1, 2, 3, ""}, false},
		{"1.2.3", Semver{1, 2, 3, ""}, false},
		{"v0.2.0", Semver{0, 2, 0, ""}, false},
		{"v1.0.0-beta", Semver{1, 0, 0, "-beta"}, false},
		{"v2.1.0+build123", Semver{2, 1, 0, "+build123"}, false},
		{"v1.2", Semver{1, 2, 0, ""}, false},
		{"bad", Semver{}, true},
		{"", Semver{}, true},
	}
	for _, c := range cases {
		got, err := ParseSemver(c.input)
		if (err != nil) != c.wantErr {
			t.Fatalf("ParseSemver(%q) err=%v, wantErr=%v", c.input, err, c.wantErr)
		}
		if err != nil {
			continue
		}
		if got != c.want {
			t.Fatalf("ParseSemver(%q) = %+v, want %+v", c.input, got, c.want)
		}
	}
}

func TestSemverIsNewerThan(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want bool
	}{
		{"v1.2.3", "v1.2.2", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.3", "v1.2.4", false},
		{"v2.0.0", "v1.9.9", true},
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0-beta", "v1.0.0", false},   // pre-release older than release
		{"v1.0.0", "v1.0.0-beta", true},      // release newer than pre-release
		{"v1.0.0-alpha", "v1.0.0-beta", false}, // alpha < beta lexicographically... wait, alpha < beta so not newer
		{"v1.0.0-beta", "v1.0.0-alpha", true},
	}
	for _, c := range cases {
		a, _ := ParseSemver(c.a)
		b, _ := ParseSemver(c.b)
		got := a.IsNewerThan(b)
		if got != c.want {
			t.Fatalf("%s.IsNewerThan(%s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestSemverString(t *testing.T) {
	sv := Semver{0, 2, 0, ""}
	if s := sv.String(); s != "v0.2.0" {
		t.Fatalf("String() = %s, want v0.2.0", s)
	}
	sv2 := Semver{1, 0, 0, "-rc1"}
	if s := sv2.String(); s != "v1.0.0-rc1" {
		t.Fatalf("String() = %s, want v1.0.0-rc1", s)
	}
}
