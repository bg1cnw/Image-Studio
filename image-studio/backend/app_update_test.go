package backend

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "major newer", a: "1.1.6", b: "0.1.5", want: 1},
		{name: "same version", a: "1.1.6", b: "1.1.6", want: 0},
		{name: "older patch", a: "1.1.5", b: "1.1.6", want: -1},
		{name: "strip v prefix", a: "v1.2.0", b: "1.1.9", want: 1},
		{name: "release beats prerelease", a: "1.2.0", b: "1.2.0-beta.1", want: 1},
		{name: "prerelease lower than release", a: "1.2.0-beta.1", b: "1.2.0", want: -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareSemver(tc.a, tc.b)
			switch {
			case tc.want > 0 && got <= 0:
				t.Fatalf("compareSemver(%q, %q) = %d, want > 0", tc.a, tc.b, got)
			case tc.want < 0 && got >= 0:
				t.Fatalf("compareSemver(%q, %q) = %d, want < 0", tc.a, tc.b, got)
			case tc.want == 0 && got != 0:
				t.Fatalf("compareSemver(%q, %q) = %d, want 0", tc.a, tc.b, got)
			}
		})
	}
}

func TestNormalizeReleaseVersion(t *testing.T) {
	if got := normalizeReleaseVersion(" v1.1.6 "); got != "1.1.6" {
		t.Fatalf("normalizeReleaseVersion() = %q, want 1.1.6", got)
	}
	if got := normalizeReleaseVersion("release-1.1.6"); got != "" {
		t.Fatalf("normalizeReleaseVersion() = %q, want empty", got)
	}
}
