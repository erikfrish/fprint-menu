package sudo

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
)

const PreserveDesktopEnv = "--preserve-env=DBUS_SESSION_BUS_ADDRESS,XDG_RUNTIME_DIR"

type Runner struct {
	Command string
	Args    []string
	Stdin   io.Reader
}

func (r Runner) Run(ctx context.Context) <-chan Event {
	events := make(chan Event, 32)
	go func() {
		defer close(events)

		name := r.Command
		if name == "" {
			name = "sudo"
		}
		cmd := exec.CommandContext(ctx, name, r.Args...)
		if r.Stdin != nil {
			cmd.Stdin = r.Stdin
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			events <- Event{Kind: EventFailed, Err: err}
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			events <- Event{Kind: EventFailed, Err: err}
			return
		}

		if err := cmd.Start(); err != nil {
			events <- Event{Kind: EventFailed, Err: err}
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go scanEvents(stdout, events, &wg)
		go scanEvents(stderr, events, &wg)
		wg.Wait()

		err = cmd.Wait()
		if errors.Is(ctx.Err(), context.Canceled) {
			events <- Event{Kind: EventFailed, Err: ctx.Err()}
			return
		}
		if err != nil {
			events <- Event{Kind: EventFailed, Err: err}
			return
		}
		events <- Event{Kind: EventSuccess}
	}()
	return events
}

func scanEvents(r io.Reader, events chan<- Event, wg *sync.WaitGroup) {
	defer wg.Done()
	parser := Parser{}
	var lastKind EventKind
	buf := make([]byte, 256)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			events <- Event{Kind: EventRaw, Line: string(buf[:n])}
			for _, event := range parser.Write(string(buf[:n])) {
				if event.Kind == lastKind && event.Kind != EventPasswordPrompt && event.Kind != EventPasswordFailed {
					continue
				}
				events <- event
				lastKind = event.Kind
			}
		}
		if err == nil {
			continue
		}
		for _, event := range parser.Close() {
			if event.Kind == lastKind && event.Kind != EventPasswordPrompt && event.Kind != EventPasswordFailed {
				continue
			}
			events <- event
			lastKind = event.Kind
		}
		if err != io.EOF {
			if strings.Contains(err.Error(), "input/output error") {
				return
			}
			events <- Event{Kind: EventFailed, Err: err}
		}
		return
	}
}

func Args(command string, args ...string) []string {
	all := []string{"-k", "-S", "-p", "", PreserveDesktopEnv, command}
	all = append(all, args...)
	return all
}
