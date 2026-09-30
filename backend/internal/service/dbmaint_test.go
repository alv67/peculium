package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/repository"
)

// recordingMaintainer is the fake DBMaintainer: it replays a canned dump
// payload and records the restores it was asked to run.
type recordingMaintainer struct {
	dumpPayload  []byte
	dumpErr      error
	restoreErr   error
	restoreCalls int
	lastPath     string
}

func (m *recordingMaintainer) Dump(ctx context.Context, w io.Writer) error {
	if m.dumpErr != nil {
		return m.dumpErr
	}
	if m.dumpPayload != nil {
		_, err := w.Write(m.dumpPayload)
		return err
	}
	return nil
}

func (m *recordingMaintainer) Restore(ctx context.Context, path string) error {
	m.restoreCalls++
	m.lastPath = path
	return m.restoreErr
}

func newSvcWithDB(db DBMaintainer) *Service {
	return New(&repository.Repository{}, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, db)
}

func TestBackupDatabase_ForwardsToMaintainer(t *testing.T) {
	m := &recordingMaintainer{dumpPayload: []byte("PGDUMP")}
	svc := newSvcWithDB(m)
	var buf bytes.Buffer
	if err := svc.BackupDatabase(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "PGDUMP" {
		t.Errorf("dumped %q, want the maintainer payload", buf.String())
	}
}

func TestBackupDatabase_NotConfigured(t *testing.T) {
	svc := newSvcWithDB(nil)
	if err := svc.BackupDatabase(context.Background(), io.Discard); !errors.Is(err, ErrDBMaintenance) {
		t.Fatalf("err = %v, want ErrDBMaintenance", err)
	}
}

func TestRestoreDatabase_RejectsMissingConfirmation(t *testing.T) {
	m := &recordingMaintainer{}
	svc := newSvcWithDB(m)
	for _, confirm := range []string{"", "yes", "replacex", "TRUE"} {
		if err := svc.RestoreDatabase(context.Background(), "/tmp/dump", confirm); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("confirm %q: err = %v, want ErrInvalidInput", confirm, err)
		}
	}
	if m.restoreCalls != 0 {
		t.Fatalf("maintainer ran %d times without confirmation", m.restoreCalls)
	}
}

func TestRestoreDatabase_AcceptsReplaceAndDelegates(t *testing.T) {
	m := &recordingMaintainer{}
	svc := newSvcWithDB(m)
	if err := svc.RestoreDatabase(context.Background(), "/tmp/dump", "REPLACE"); err != nil {
		t.Fatalf("case-insensitive confirm rejected: %v", err)
	}
	if m.restoreCalls != 1 || m.lastPath != "/tmp/dump" {
		t.Fatalf("calls = %d path = %q, want 1 /tmp/dump", m.restoreCalls, m.lastPath)
	}
}

func TestRestoreDatabase_NotConfigured(t *testing.T) {
	svc := newSvcWithDB(nil)
	if err := svc.RestoreDatabase(context.Background(), "/tmp/dump", DBRestoreConfirm); !errors.Is(err, ErrDBMaintenance) {
		t.Fatalf("err = %v, want ErrDBMaintenance", err)
	}
}

func TestRestoreDatabase_ForwardsMaintainerError(t *testing.T) {
	boom := errors.New("pg_restore failed: boom")
	m := &recordingMaintainer{restoreErr: boom}
	svc := newSvcWithDB(m)
	if err := svc.RestoreDatabase(context.Background(), "/tmp/dump", DBRestoreConfirm); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the maintainer error", err)
	}
}

func writeStubTool(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stub tools are POSIX")
	}
	path := filepath.Join(t.TempDir(), "stub.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExecDump_StreamsStdoutAndPassesCredentialsViaEnv(t *testing.T) {
	tool := writeStubTool(t, `printf 'args:%s|pw:%s|ssl:%s\n' "$*" "$PGPASSWORD" "$PGSSLMODE"`)
	m := NewExecDBMaintainer(DBConnection{Host: "dbhost", Port: 5433, User: "dbuser", Name: "dbname", Password: "dbpass", SSLMode: "require"})
	m.dumpBin = tool

	var out bytes.Buffer
	if err := m.Dump(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"-Fc", "--no-password", "-h dbhost", "-p 5433", "-U dbuser", "dbname"} {
		if !strings.Contains(got, want) {
			t.Errorf("pg_dump invocation missing %q in %q", want, got)
		}
	}
	// The password must reach the tool through the environment only.
	argsPart := got[strings.Index(got, "args:"):strings.Index(got, "|pw:")]
	if strings.Contains(argsPart, "dbpass") {
		t.Errorf("password leaked into the command line: %q", got)
	}
	if !strings.Contains(got, "pw:dbpass") || !strings.Contains(got, "ssl:require") {
		t.Errorf("PGPASSWORD/PGSSLMODE not propagated: %q", got)
	}
}

func TestExecRestore_PassesDestructiveFlagsAndMasksSecretInErrors(t *testing.T) {
	tool := writeStubTool(t, `echo "args: $*" >&2; echo "auth: $PGPASSWORD" >&2; exit 1`)
	m := NewExecDBMaintainer(DBConnection{Host: "h", Port: 5432, User: "u", Name: "peculium", Password: "TOPSECRET", SSLMode: "disable"})
	m.restoreBin = tool

	err := m.Restore(context.Background(), "/tmp/dump")
	if !errors.Is(err, ErrDBMaintenance) {
		t.Fatalf("err = %v, want ErrDBMaintenance", err)
	}
	msg := err.Error()
	for _, want := range []string{"--clean", "--if-exists", "--no-owner", "--no-acl", "-d peculium", "/tmp/dump"} {
		if !strings.Contains(msg, want) {
			t.Errorf("pg_restore invocation missing %q in %q", want, msg)
		}
	}
	if strings.Contains(msg, "TOPSECRET") {
		t.Errorf("sanitized error still leaks the password: %q", msg)
	}
	if !strings.Contains(msg, "*****") {
		t.Errorf("sanitized error should mask the password: %q", msg)
	}
}

func TestExec_MissingBinaryFailsAsMaintenanceError(t *testing.T) {
	m := NewExecDBMaintainer(DBConnection{Name: "dbname"})
	m.dumpBin = filepath.Join(t.TempDir(), "definitely-missing")
	var out bytes.Buffer
	if err := m.Dump(context.Background(), &out); !errors.Is(err, ErrDBMaintenance) {
		t.Fatalf("err = %v, want ErrDBMaintenance", err)
	}
}

func TestCappedWriter_Truncates(t *testing.T) {
	w := &cappedWriter{max: 5}
	_, _ = w.Write([]byte("abcdefgh"))
	if w.buf.String() != "abcde" {
		t.Errorf("kept %q, want the first 5 bytes", w.buf.String())
	}
}

func TestSanitizeSecret(t *testing.T) {
	if got := sanitizeSecret("user=u password=sekret db=d", "sekret"); strings.Contains(got, "sekret") || !strings.Contains(got, "*****") {
		t.Errorf("not masked: %q", got)
	}
	if got := sanitizeSecret("no secret here", ""); got != "no secret here" {
		t.Errorf("empty-secret sanitize changed text: %q", got)
	}
}
