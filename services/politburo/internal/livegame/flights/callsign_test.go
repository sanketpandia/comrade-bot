package flights

import "testing"

func TestNormalizedCallsignForAffix(t *testing.T) {
	cases := map[string]string{
		"Delta 095AK Heavy":         "Delta 095AK",
		"Airline 001VA Flight of 5": "Airline 001VA",
		"SW39 Heavy Flight of 2":    "SW39",
		"IE123":                     "IE123",
	}
	for in, want := range cases {
		if got := NormalizedCallsignForAffix(in); got != want {
			t.Fatalf("NormalizedCallsignForAffix(%q) = %q, want %q", in, got, want)
		}
	}
}
