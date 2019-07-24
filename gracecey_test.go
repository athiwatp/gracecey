package gracecey

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"
)

type testServer struct {
	shutdown bool
	wait     time.Duration
}

func (t *testServer) Shutdown(ctx context.Context) error {
	wait := time.NewTimer(t.wait)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait.C:
		// server finished fake busy work
	}

	t.shutdown = true

	return nil
}

type logRecorder struct {
	infos  []string
	errors []string
}

func (l *logRecorder) Infof(format string, args ...interface{}) {
	l.infos = append(l.infos, fmt.Sprintf(format, args...))
}

func (l *logRecorder) Errorf(format string, args ...interface{}) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func Test(t *testing.T) {
	srv := &testServer{wait: time.Second}
	log := &logRecorder{}

	graceful := &FlushFinish{Log: log}
	graceful.Add(srv)

	keeper := graceful.keepers[0]

	if keeper.srv != srv {
		t.Error("expected server to be added")
	}

	if keeper.name != "" {
		t.Error("expected name to be empty")
	}

	if keeper.timeout != DefaultTimeout {
		t.Error("expected timeout to be the default")
	}

	go graceful.Trigger()

	if srv.shutdown {
		t.Error("expected server not to be shutdown yet")
	}

	graceful.Wait()

	if !srv.shutdown {
		t.Error("expected server to be shutdown")
	}

	if keeper.name != "server" {
		t.Error("expected name to be 'server'")
	}

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
		"finish: server closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}

func TestSettingName(t *testing.T) {
	srv := &testServer{}
	log := &logRecorder{}

	graceful := &FlushFinish{Log: log}
	graceful.Add(srv, WithName("foobar"))

	keeper := graceful.keepers[0]

	if keeper.name != "foobar" {
		t.Error("expected name to be set")
	}

	go graceful.Trigger()

	graceful.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down foobar ...",
		"finish: foobar closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}

func TestUpdateNames(t *testing.T) {
	srv := &testServer{}

	graceful := New()
	graceful.Add(srv, WithName("foobar"))
	graceful.Add(srv)

	graceful.updateNames()

	if graceful.keepers[0].name != "foobar" {
		t.Error("wrong name")
	}

	if graceful.keepers[1].name != "server #2" {
		t.Error("wrong name")
	}
}

func TestGlobalTimeout(t *testing.T) {
	srv := &testServer{}

	graceful := &FlushFinish{Timeout: 21 * time.Second}
	graceful.Add(srv)

	keeper := graceful.keepers[0]

	if keeper.timeout != 21*time.Second {
		t.Error("expected timeout to be changed")
	}
}

func TestOverridingTimeout(t *testing.T) {
	srv := &testServer{}

	graceful := New()
	graceful.Add(srv, WithTimeout(42*time.Second))

	keeper := graceful.keepers[0]

	if keeper.timeout != 42*time.Second {
		t.Error("expected timeout to be set")
	}
}

func TestSlowServer(t *testing.T) {
	srv := &testServer{wait: 2 * time.Second}
	log := &logRecorder{}

	graceful := &FlushFinish{Log: log}
	graceful.Add(srv, WithTimeout(time.Second))

	go graceful.Trigger()

	graceful.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
	}) {
		t.Error("wrong log output")
	}

	if !reflect.DeepEqual(log.errors, []string{
		"finish: shutdown timeout for server",
	}) {
		t.Error("wrong error log output")
	}
}

func TestCustomSignal(t *testing.T) {
	srv := &testServer{}
	log := &logRecorder{}

	mySignal := syscall.SIGUSR1

	graceful := &FlushFinish{Log: log, Signals: []os.Signal{mySignal}}
	graceful.Add(srv)

	go func() {
		// sleep so Wait() can actually catch the signal
		time.Sleep(time.Second)
		// trigger custom signal
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		p.Signal(mySignal)
	}()

	graceful.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
		"finish: server closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}
