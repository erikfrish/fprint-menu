package sudo

import "strings"

type EventKind int

const (
	EventUnknown EventKind = iota
	EventPasswordPrompt
	EventFingerprintPrompt
	EventFingerprintFailed
	EventPasswordFailed
	EventNoPassword
	EventSuccess
	EventFailed
	EventRaw
)

type Event struct {
	Kind EventKind
	Line string
	Err  error
}

type Parser struct {
	buf      string
	lastKind EventKind
	emitted  string
}

func (p *Parser) Write(data string) []Event {
	p.buf += data
	return p.flush(false)
}

func (p *Parser) Close() []Event {
	return p.flush(true)
}

func (p *Parser) flush(final bool) []Event {
	var events []Event
	for {
		idx := strings.IndexByte(p.buf, '\n')
		if idx < 0 {
			break
		}
		line := p.buf[:idx]
		p.buf = p.buf[idx+1:]
		line = strings.TrimRight(line, "\r")
		if event, ok := ParseLine(line); ok {
			events = append(events, event)
			p.lastKind = event.Kind
			p.emitted = ""
		}
	}

	trimmed := strings.TrimSpace(p.buf)
	if trimmed != "" {
		if event, ok := ParseLine(p.buf); ok && event.Kind != EventUnknown {
			if trimmed != p.emitted {
				events = append(events, event)
				p.lastKind = event.Kind
				p.emitted = trimmed
				p.buf = ""
			}
		}
	}

	if final {
		p.buf = ""
		p.emitted = ""
	}
	return events
}

func ParseEvents(output string) []Event {
	var events []Event
	for _, line := range strings.Split(output, "\n") {
		if event, ok := ParseLine(line); ok {
			events = append(events, event)
		}
	}
	return events
}

func ParseLine(line string) (Event, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Event{}, false
	}
	text := strings.ToLower(trimmed)
	switch {
	case strings.Contains(text, "failed to match fingerprint"):
		return Event{Kind: EventFingerprintFailed, Line: trimmed}, true
	case strings.Contains(text, "place your finger") || strings.Contains(text, "fingerprint reader") || strings.Contains(text, "reader again"):
		return Event{Kind: EventFingerprintPrompt, Line: trimmed}, true
	case strings.Contains(text, "no password was provided"):
		return Event{Kind: EventNoPassword, Line: trimmed}, true
	case strings.Contains(text, "sorry, try again") || strings.Contains(text, "incorrect password"):
		return Event{Kind: EventPasswordFailed, Line: trimmed}, true
	case strings.Contains(text, "password"):
		return Event{Kind: EventPasswordPrompt, Line: trimmed}, true
	default:
		return Event{Kind: EventUnknown, Line: trimmed}, true
	}
}
