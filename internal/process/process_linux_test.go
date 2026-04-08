// +build linux

package process

import "testing"

func TestParseProcStat(t *testing.T) {
	comm, startTicks, err := parseProcStat([]byte("1234 (python3) S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 424242"))
	if err != nil {
		t.Fatal(err)
	}

	if comm != "python3" {
		t.Fatalf("unexpected comm: %q", comm)
	}
	if startTicks != 424242 {
		t.Fatalf("unexpected start ticks: %d", startTicks)
	}
}

func TestParseProcStatRejectsInvalidInput(t *testing.T) {
	_, _, err := parseProcStat([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error")
	}
}
