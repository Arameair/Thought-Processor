package tass

// Stage 3B runtime repair (Codex audit, PR #122 comment 6850390555): Core's experiment receipts are a
// trust boundary, checked every time one is read, shown, built on or written beside.
//
// A receipt (<data>/experiment-runs/<exp>/snapshot.json, A|B.submitted.json, A|B.outcome.json) is trusted
// only if, at the moment of use:
//   - experiment-runs/ and experiment-runs/<exp>/ are real directories (never symlinks) with no
//     permission beyond their owner's (nothing outside 0700);
//   - the file is regular (never a symlink, directory or device) with no permission beyond owner
//     read/write (0400 and 0600 are accepted), and no larger than expReceiptMax;
//   - it is exactly one JSON value followed by one newline, with this schema.
// The checks are made on the path (Lstat) and on the opened file (opened without following a symlink,
// Fstat, compared with the Lstat result), with a bounded read. Nothing is repaired, chmod-ed or
// re-created: an unsafe receipt is left as found, its arm is RECOVERY_REQUIRED (no reply shown) or the
// snapshot UNVERIFIED, and nothing further is sent. Identity checks (owner, Engine entity, Thought, arm,
// variant, snapshot, approval, reply digest) are in experiment_run.go (trustedArm, trustedSnapshot).
//
// On Windows the permission bits Go reports do not describe access, so only the type, symlink, size and
// bounded-read rules apply there. The supported deployment (Linux/WSL) enforces all of them.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// One reply of at most 256 tokens plus receipts and IDs is a few KiB.
const expReceiptMax = 256 << 10

var (
	errExpUnsafe = errors.New("not a private, regular, bounded file in private directories")
	hex64RE      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func expPrivateDir(path string) (exists bool, err error) {
	fi, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return true, fmt.Errorf("%w (%s cannot be inspected)", errExpUnsafe, filepath.Base(path))
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return true, fmt.Errorf("%w (%s is not a real directory)", errExpUnsafe, filepath.Base(path))
	}
	if !expModeOK(fi.Mode(), 0o700) {
		return true, fmt.Errorf("%w (directory %s has permissions %04o; nothing beyond 0700 is accepted)", errExpUnsafe, filepath.Base(path), fi.Mode().Perm())
	}
	return true, nil
}

// expTree checks (and with create, makes one level at a time, 0700) experiment-runs/ and
// experiment-runs/<exp>/. exists=false: a level is absent and create was false.
func (c *Core) expTree(exp string, create bool) (exists bool, err error) {
	if !experimentIDRE.MatchString(exp) {
		return false, fmt.Errorf("%w (not an experiment ID)", errExpUnsafe)
	}
	if fi, lerr := os.Lstat(c.cfg.DataDir); lerr != nil || !fi.IsDir() {
		return false, fmt.Errorf("%w (the data folder is not a directory)", errExpUnsafe)
	}
	dir := c.cfg.DataDir
	for _, l := range []string{experimentRunDir, exp} {
		dir = filepath.Join(dir, l)
		ex, derr := expPrivateDir(dir)
		if derr != nil {
			return true, derr
		}
		if ex {
			continue
		}
		if !create {
			return false, nil
		}
		if merr := os.Mkdir(dir, 0o700); merr != nil && !errors.Is(merr, os.ErrExist) {
			return false, fmt.Errorf("%w (%s cannot be created)", errExpUnsafe, l)
		}
		if ex, derr := expPrivateDir(dir); derr != nil || !ex {
			if derr == nil {
				derr = fmt.Errorf("%w (%s disappeared)", errExpUnsafe, l)
			}
			return true, derr
		}
	}
	return true, nil
}

func expPrivateRead(path string, max int64) (b []byte, exists bool, err error) {
	li, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("%w (%s cannot be inspected)", errExpUnsafe, filepath.Base(path))
	}
	check := func(fi os.FileInfo) error {
		switch {
		case !fi.Mode().IsRegular():
			return fmt.Errorf("%w (%s is not a regular file)", errExpUnsafe, filepath.Base(path))
		case !expModeOK(fi.Mode(), 0o600):
			return fmt.Errorf("%w (%s has permissions %04o; nothing beyond 0600 is accepted)", errExpUnsafe, filepath.Base(path), fi.Mode().Perm())
		case fi.Size() > max:
			return fmt.Errorf("%w (%s is %d bytes; the limit is %d)", errExpUnsafe, filepath.Base(path), fi.Size(), max)
		}
		return nil
	}
	if err := check(li); err != nil {
		return nil, true, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|expOpenNoFollow, 0)
	if err != nil {
		return nil, true, fmt.Errorf("%w (%s cannot be opened without following a link)", errExpUnsafe, filepath.Base(path))
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !os.SameFile(li, fi) {
		return nil, true, fmt.Errorf("%w (%s changed while it was being opened)", errExpUnsafe, filepath.Base(path))
	}
	if err := check(fi); err != nil {
		return nil, true, err
	}
	b, err = io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, true, fmt.Errorf("%w (%s cannot be read)", errExpUnsafe, filepath.Base(path))
	}
	if int64(len(b)) > max {
		return nil, true, fmt.Errorf("%w (%s grew past the limit of %d bytes)", errExpUnsafe, filepath.Base(path), max)
	}
	return b, true, nil
}

// expRead reads one receipt of experiment exp. exists=true, ok=false: something is there (the file, or an
// unsafe directory above it) that cannot be trusted.
func (c *Core) expRead(exp, name string, v any) (exists, ok bool) {
	treeExists, err := c.expTree(exp, false)
	if err != nil {
		return true, false
	}
	if !treeExists {
		return false, false
	}
	b, exists, err := expPrivateRead(c.expRunPath(exp, name), expReceiptMax)
	if !exists {
		return false, false
	}
	if err != nil || len(b) < 2 || b[len(b)-1] != '\n' {
		return true, false
	}
	dec := json.NewDecoder(bytes.NewReader(b[:len(b)-1]))
	if dec.Decode(v) != nil || dec.More() {
		return true, false
	}
	var s struct {
		Schema string `json:"schema"`
	}
	json.Unmarshal(b, &s)
	return true, s.Schema == experimentRunSchema
}

// expCreate writes one receipt once (O_EXCL, 0600, fsync) inside the checked private tree.
func (c *Core) expCreate(exp, name string, v any) (bool, error) {
	if _, err := c.expTree(exp, true); err != nil {
		return false, err
	}
	return coreCreateOnce(c.expRunPath(exp, name), v)
}
