package build

import "testing"

func TestHostSupportsVncRecording(t *testing.T) {
	testCases := []struct {
		goos string
		want bool
	}{
		{goos: "darwin", want: true},
		{goos: "linux", want: true},
		{goos: "windows", want: false},
	}

	for _, tc := range testCases {
		if got := hostSupportsVncRecording(tc.goos); got != tc.want {
			t.Fatalf("hostSupportsVncRecording(%q) = %v, want %v", tc.goos, got, tc.want)
		}
	}
}

func TestHostSupportsVncViewer(t *testing.T) {
	testCases := []struct {
		goos string
		want bool
	}{
		{goos: "darwin", want: true},
		{goos: "linux", want: false},
		{goos: "windows", want: false},
	}

	for _, tc := range testCases {
		if got := hostSupportsVncViewer(tc.goos); got != tc.want {
			t.Fatalf("hostSupportsVncViewer(%q) = %v, want %v", tc.goos, got, tc.want)
		}
	}
}
