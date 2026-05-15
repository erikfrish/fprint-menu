package sudo

import "testing"

func TestParseEvents(t *testing.T) {
	tests := []struct {
		name string
		line string
		want EventKind
	}{
		{name: "password requested immediately", line: "[sudo] password for user:", want: EventPasswordPrompt},
		{name: "fingerprint requested first", line: "Place your finger on the fingerprint reader", want: EventFingerprintPrompt},
		{name: "fingerprint requested again", line: "Place your finger on the reader again", want: EventFingerprintPrompt},
		{name: "fingerprint did not match", line: "Failed to match fingerprint", want: EventFingerprintFailed},
		{name: "no password provided", line: "sudo: no password was provided", want: EventNoPassword},
		{name: "wrong password", line: "Sorry, try again.", want: EventPasswordFailed},
		{name: "incorrect password attempt", line: "sudo: 1 incorrect password attempt", want: EventPasswordFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, ok := ParseLine(tt.line)
			if !ok {
				t.Fatal("expected event")
			}
			if event.Kind != tt.want {
				t.Fatalf("ParseLine() = %v, want %v", event.Kind, tt.want)
			}
		})
	}
}

func TestParseObservedSudoPamTrace(t *testing.T) {
	trace := `Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
[sudo] password for erikfrish:
Sorry, try again.
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
[sudo] password for erikfrish:
Sorry, try again.
Place your finger on the fingerprint reader`

	want := []EventKind{
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventPasswordPrompt,
		EventPasswordFailed,
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventFingerprintPrompt,
		EventFingerprintFailed,
		EventPasswordPrompt,
		EventPasswordFailed,
		EventFingerprintPrompt,
	}

	got := ParseEvents(trace)
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Kind != want[i] {
			t.Fatalf("event %d = %v, want %v", i, got[i].Kind, want[i])
		}
	}
}

func TestParserEmitsPasswordPromptWithoutNewline(t *testing.T) {
	var parser Parser
	events := parser.Write("[sudo] password for erikfrish:")
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Kind != EventPasswordPrompt {
		t.Fatalf("event = %v, want %v", events[0].Kind, EventPasswordPrompt)
	}
}

func TestParserEmitsFingerprintPromptWithoutNewline(t *testing.T) {
	var parser Parser
	events := parser.Write("Place your finger on the fingerprint reader")
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Kind != EventFingerprintPrompt {
		t.Fatalf("event = %v, want %v", events[0].Kind, EventFingerprintPrompt)
	}
}

func TestParserDoesNotRepeatSamePartialPrompt(t *testing.T) {
	var parser Parser
	first := parser.Write("[sudo] password for erikfrish:")
	if len(first) != 1 {
		t.Fatalf("first write events = %d, want 1", len(first))
	}
	second := parser.Write("[sudo] password for erikfrish:")
	if len(second) != 0 {
		t.Fatalf("second write same prompt events = %d, want 0", len(second))
	}
}

func TestParserDoesNotReemitOnEcho(t *testing.T) {
	var parser Parser
	first := parser.Write("[sudo] password for erikfrish: ")
	if len(first) != 1 || first[0].Kind != EventPasswordPrompt {
		t.Fatalf("first write: got %d events, want 1 password prompt", len(first))
	}
	second := parser.Write("\r\n")
	if len(second) != 0 {
		t.Fatalf("echo newline events = %d, want 0", len(second))
	}
}
