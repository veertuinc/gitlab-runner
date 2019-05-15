package common

import (
	"context"
	"io"
	"os"
	"sync"
)

type Trace struct {
	Writer     io.Writer
	CancelFunc context.CancelFunc
	mutex      sync.Mutex
}

func (s *Trace) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Writer == nil {
		return 0, os.ErrInvalid
	}
	return s.Writer.Write(p)
}

func (s *Trace) SetMasked(values []string) {
}

func (s *Trace) Success() {
}

func (s *Trace) Fail(err error, failureReason JobFailureReason) {
}

func (s *Trace) SetCancelFunc(cancelFunc context.CancelFunc) {
	s.CancelFunc = cancelFunc
}

func (s *Trace) SetFailuresCollector(fc FailuresCollector) {}

func (s *Trace) IsStdout() bool {
	return true
}
