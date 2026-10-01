package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// DBRestoreConfirm is the confirmation value a restore request must carry:
// replacing the whole database is destructive and never happens without it.
const DBRestoreConfirm = "replace"

// dbStderrLimit caps how much pg_dump/pg_restore stderr is kept in errors.
const dbStderrLimit = 4096

// DBMaintainer runs the database-level maintenance commands behind the admin
// backup/restore endpoints: Dump writes a full custom-format archive to w
// and Restore replaces the database from the archive stored at path. The
// interface exists so handlers can be tested with a fake.
type DBMaintainer interface {
	Dump(ctx context.Context, w io.Writer) error
	Restore(ctx context.Context, path string) error
}

// DBConnection carries the database coordinates the CLI tools connect with.
// The values come from config.Config; the password never reaches the command
// line, it is passed to the client tools through the PGPASSWORD environment
// variable, and the TLS setting through PGSSLMODE.
type DBConnection struct {
	Host     string
	Port     int
	User     string
	Name     string
	Password string
	SSLMode  string
}

// ExecDBMaintainer shells out to the PostgreSQL client tools shipped in the
// server image (pg_dump / pg_restore, custom format -Fc). dumpBin and
// restoreBin are overridable seams for tests.
type ExecDBMaintainer struct {
	db         DBConnection
	dumpBin    string
	restoreBin string
}

func NewExecDBMaintainer(db DBConnection) *ExecDBMaintainer {
	return &ExecDBMaintainer{db: db, dumpBin: "pg_dump", restoreBin: "pg_restore"}
}

func (m *ExecDBMaintainer) env() []string {
	return append(os.Environ(),
		"PGPASSWORD="+m.db.Password,
		"PGSSLMODE="+m.db.SSLMode,
	)
}

func (m *ExecDBMaintainer) Dump(ctx context.Context, w io.Writer) error {
	args := []string{"-Fc", "--no-password", "-h", m.db.Host, "-p", strconv.Itoa(m.db.Port), "-U", m.db.User, m.db.Name}
	return m.run(ctx, "pg_dump", m.dumpBin, args, m.db.Password, w)
}

func (m *ExecDBMaintainer) Restore(ctx context.Context, path string) error {
	args := []string{"--clean", "--if-exists", "--no-owner", "--no-acl", "--no-password",
		"-h", m.db.Host, "-p", strconv.Itoa(m.db.Port), "-U", m.db.User, "-d", m.db.Name, path}
	return m.run(ctx, "pg_restore", m.restoreBin, args, m.db.Password, nil)
}

// run executes one client tool with its stderr captured (capped) and folded
// into the returned error with the password masked, so the admin can see why
// a command failed without the dump leaking the credential through a tool
// message that echoes the connection attempt. A non-nil stdout receives the
// tool's standard output.
func (m *ExecDBMaintainer) run(ctx context.Context, tool, bin string, args []string, secret string, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = m.env()
	stderr := &cappedWriter{max: dbStderrLimit}
	cmd.Stderr = stderr
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if err := cmd.Run(); err != nil {
		msg := sanitizeSecret(strings.TrimSpace(stderr.String()), secret)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%w: %s failed: %s", ErrDBMaintenance, tool, msg)
	}
	return nil
}

// cappedWriter keeps at most max bytes of what is written to it, so a
// chatty failing tool cannot grow an unbounded error string.
type cappedWriter struct {
	max int
	buf bytes.Buffer
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	if remaining := w.max - w.buf.Len(); remaining > 0 {
		if len(p) > remaining {
			w.buf.Write(p[:remaining])
			return len(p), nil
		}
		w.buf.Write(p)
	}
	return len(p), nil
}

func (w *cappedWriter) String() string { return w.buf.String() }

// sanitizeSecret masks every occurrence of the database password (and the
// empty-trimmed no-op case) in text coming from an external tool.
func sanitizeSecret(text, secret string) string {
	if secret == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, "*****")
}

// BackupDatabase streams a full custom-format dump of the application
// database into w.
func (s *Service) BackupDatabase(ctx context.Context, w io.Writer) error {
	return s.db.Dump(ctx, w)
}

// RestoreDatabase replaces the whole database from the custom-format archive
// stored at path. confirm must be exactly DBRestoreConfirm: the operation
// drops and recreates every table, including the users, so it never runs on
// a request that did not explicitly ask for it.
func (s *Service) RestoreDatabase(ctx context.Context, path, confirm string) error {
	if !strings.EqualFold(strings.TrimSpace(confirm), DBRestoreConfirm) {
		return fmt.Errorf("%w: restoring requires the explicit confirmation %q", ErrInvalidInput, DBRestoreConfirm)
	}
	return s.db.Restore(ctx, path)
}
