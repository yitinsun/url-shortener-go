package encoding

import (
	"math"
	"strings"
	"testing"
)

func TestPivotIDIsFiveToSevenAlphanumericChars(t *testing.T) {
	encoded := Encode(56800235)
	if len(encoded) < 5 || len(encoded) > 7 {
		t.Fatalf("length %d for %q", len(encoded), encoded)
	}
	for _, ch := range encoded {
		if !isAlphaNum(ch) {
			t.Fatalf("non-alphanumeric %q in %q", ch, encoded)
		}
	}
	if strings.ContainsAny(encoded, "+/") {
		t.Fatalf("alphabet leaked: %q", encoded)
	}
	got, err := Decode(encoded)
	if err != nil || got != 56800235 {
		t.Fatalf("decode %q => %d, %v", encoded, got, err)
	}
}

func TestAlphabetExcludesBase64Symbols(t *testing.T) {
	for _, id := range []uint64{1, 61, 62, 63, 1_000_000, 56_800_235} {
		encoded := Encode(id)
		if strings.ContainsAny(encoded, "+/") {
			t.Fatalf("%d encoded as %q", id, encoded)
		}
	}
}

func TestMaxInt64RoundTripsWithoutOverflow(t *testing.T) {
	id := uint64(math.MaxInt64)
	encoded := Encode(id)
	got, err := Decode(encoded)
	if err != nil || got != id {
		t.Fatalf("round trip %d via %q => %d, %v", id, encoded, got, err)
	}
}

func TestRandomPositiveIDsRoundTrip(t *testing.T) {
	state := uint64(0x123456789abcdef0)
	for i := 0; i < 32; i++ {
		state = state*6364136223846793005 + 1
		id := state & uint64(math.MaxInt64)
		got, err := Decode(Encode(id))
		if err != nil || got != id {
			t.Fatalf("id %d => %d, %v", id, got, err)
		}
	}
}

func TestZeroEncodesAsZero(t *testing.T) {
	if Encode(0) != "0" {
		t.Fatalf("encode(0) = %q", Encode(0))
	}
	got, err := Decode("0")
	if err != nil || got != 0 {
		t.Fatalf("decode 0 => %d, %v", got, err)
	}
}

func TestDecodeRejectsBlankUnknownAndOverflowingValues(t *testing.T) {
	if _, err := Decode(""); err == nil {
		t.Fatal("blank should fail")
	}
	if _, err := Decode("abc-"); err == nil {
		t.Fatal("dash should fail")
	}
	if _, err := Decode(strings.Repeat("z", 40)); err == nil {
		t.Fatal("overflow should fail")
	}
}

func isAlphaNum(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
