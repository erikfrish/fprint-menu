package sudo

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type Session struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	events chan Event
	once   sync.Once
}

func Start(ctx context.Context, args ...string) (*Session, error) {
	cmd := exec.CommandContext(ctx, "sudo", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	s := &Session{cmd: cmd, stdin: stdin, events: make(chan Event, 32)}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scanEvents(stdout, s.events, &wg)
	go scanEvents(stderr, s.events, &wg)
	go func() {
		wg.Wait()
		err := cmd.Wait()
		if errors.Is(ctx.Err(), context.Canceled) {
			s.events <- Event{Kind: EventFailed, Err: ctx.Err()}
		} else if err != nil {
			s.events <- Event{Kind: EventFailed, Err: err}
		} else {
			s.events <- Event{Kind: EventSuccess}
		}
		close(s.events)
	}()
	return s, nil
}

func StartValidate(ctx context.Context) (*Session, error) {
	return Start(ctx, "-k", "-S", "-p", "", "-v")
}

func StartValidatePTY(ctx context.Context) (*Session, error) {
	cmd := exec.CommandContext(ctx, "sudo", "-k", "-v")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	s := &Session{cmd: cmd, stdin: ptmx, events: make(chan Event, 32)}
	var wg sync.WaitGroup
	wg.Add(1)
	go scanEvents(ptmx, s.events, &wg)
	go func() {
		wg.Wait()
		_ = ptmx.Close()
		err := cmd.Wait()
		if errors.Is(ctx.Err(), context.Canceled) {
			s.events <- Event{Kind: EventFailed, Err: ctx.Err()}
		} else if err != nil {
			s.events <- Event{Kind: EventFailed, Err: err}
		} else {
			s.events <- Event{Kind: EventSuccess}
		}
		close(s.events)
	}()
	return s, nil
}

func (s *Session) Events() <-chan Event {
	return s.events
}

func (s *Session) SendPassword(password string) error {
	_, err := io.WriteString(s.stdin, password+"\n")
	return err
}

func (s *Session) CloseStdin() error {
	return s.stdin.Close()
}
