package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"unbalance/daemon/common"
	"unbalance/daemon/domain"
)

type checkedEntry struct {
	path string
	info os.FileInfo
}

// Inspect the whole GATHER batch before the first copy or deletion. Validating
// only the current source would allow a later command to overwrite a version
// whose source had already been removed. Filters never hide such conflicts.
func validateGatherConflicts(ctx context.Context, commands []*domain.Command) error {
	seen := map[string]checkedEntry{}
	parents := map[string]bool{}
	for _, command := range commands {
		paths, err := safeCommandPaths(command)
		if err != nil {
			return err
		}
		if err = validateExistingPathBoundary(paths.SrcRoot, paths.SrcPath); err != nil {
			return err
		}
		if err = validateCreatablePathBoundary(paths.DstRoot, paths.DstPath); err != nil {
			return err
		}
		err = filepath.WalkDir(paths.SrcPath, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			rel, err := filepath.Rel(paths.SrcPath, path)
			if err != nil {
				return err
			}
			target := filepath.Join(paths.DstPath, rel)
			info, err := entry.Info()
			if err != nil {
				return err
			}
			current := checkedEntry{path, info}
			if !info.IsDir() && parents[target] {
				return fmt.Errorf("Conflitto GATHER file/cartella: %s; nessuna sorgente rimossa", command.Entry)
			}
			for parent := filepath.Dir(target); pathIsInside(paths.DstRoot, parent) && parent != paths.DstRoot; parent = filepath.Dir(parent) {
				if earlier, ok := seen[parent]; ok && !earlier.info.IsDir() {
					return fmt.Errorf("Conflitto GATHER file/cartella: %s; nessuna sorgente rimossa", command.Entry)
				}
				parents[parent] = true
			}
			if earlier, ok := seen[target]; ok {
				equal, err := sameEntryContent(ctx, earlier, current)
				if err != nil {
					return err
				}
				if !equal {
					return fmt.Errorf("Conflitto GATHER tra sorgenti: %s; nessuna sorgente rimossa", command.Entry)
				}
			} else {
				seen[target] = current
			}
			other, err := os.Lstat(target)
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			equal, err := sameEntryContent(ctx, current, checkedEntry{target, other})
			if err != nil {
				return err
			}
			if !equal {
				return fmt.Errorf("Conflitto GATHER con destinazione: %s; nessuna sorgente rimossa", command.Entry)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

func sameEntryContent(ctx context.Context, a, b checkedEntry) (bool, error) {
	if a.info.Mode().Type() != b.info.Mode().Type() {
		return false, nil
	}
	if a.info.IsDir() {
		return true, nil
	}
	if a.info.Mode()&os.ModeSymlink != 0 {
		first, err := os.Readlink(a.path)
		if err != nil {
			return false, err
		}
		second, err := os.Readlink(b.path)
		return first == second, err
	}
	if !a.info.Mode().IsRegular() {
		return false, fmt.Errorf("Tipo di file non verificabile nel confronto GATHER")
	}
	if a.info.Size() != b.info.Size() {
		return false, nil
	}
	first, err := entryDigest(ctx, a.path)
	if err != nil {
		return false, err
	}
	second, err := entryDigest(ctx, b.path)
	return first == second, err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func entryDigest(ctx context.Context, path string) ([32]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, contextReader{ctx, file}); err != nil {
		return [32]byte{}, err
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

// A directory existing on another disk is normal for split shares. Only
// overlapping non-directory relative paths can shadow a transferred file.
func transferHasOverlap(ctx context.Context, src, other string) (bool, error) {
	overlap := false
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		_, err = os.Lstat(filepath.Join(other, rel))
		if err == nil {
			overlap = true
			return filepath.SkipAll
		}
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	})
	return overlap, err
}

// Quiet is a switch only when it is not an argument to another short option.
// In attached filter/path/block-size/remote arguments every byte is data.
func comparisonFlags(flags []string) []string {
	out := []string{}
	takesNext := false
	const withArgument = "fTBMe@" // conservative: never rewrite value-bearing groups
	for _, f := range flags {
		if takesNext {
			out = append(out, f)
			takesNext = false
			continue
		}
		if f == "--quiet" || f == "--msgs2stderr" || f == "--no-msgs2stderr" {
			continue
		}
		if f == "-q" {
			continue
		}
		if strings.HasPrefix(f, "--") {
			out = append(out, f)
			// Preserve standalone values of the options supported by this UI.
			switch f {
			case "--filter", "--include", "--exclude", "--include-from", "--exclude-from", "--files-from", "--temp-dir", "--block-size", "--remote-option", "--out-format", "--log-file-format", "--info", "--debug", "--chmod", "--bwlimit", "--max-size", "--min-size":
				takesNext = true
			}
			continue
		}
		if strings.HasPrefix(f, "-") {
			group := f[1:]
			letters := len(group) > 0
			argumentAt := -1
			for i, r := range group {
				if r < 'A' || r > 'Z' && r < 'a' || r > 'z' {
					letters = false
				}
				if argumentAt < 0 && strings.ContainsRune(withArgument, r) {
					argumentAt = i
				}
			}
			if argumentAt >= 0 {
				takesNext = argumentAt == len(group)-1
			} else if letters {
				f = "-" + strings.ReplaceAll(group, "q", "")
				if f == "-" {
					continue
				}
			}
		}
		out = append(out, f)
	}
	return out
}

// This is the transfer sequencer used by runOperation, also exercised with
// real rsync and deletion callbacks on isolated fixtures.
func runTransferBatch(ctx context.Context, operation *domain.Operation, execute func(*domain.Command) error) (*domain.Command, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if operation.OpKind == common.OpGatherMove {
		if err := validateGatherConflicts(ctx, operation.Commands); err != nil {
			return nil, err
		}
	}
	for _, command := range operation.Commands {
		if err := ctx.Err(); err != nil {
			return command, err
		}
		if operation.OpKind == common.OpGatherMove {
			if err := validateGatherConflicts(ctx, []*domain.Command{command}); err != nil {
				return command, err
			}
		}
		if err := execute(command); err != nil {
			return command, err
		}
	}
	return nil, nil
}
