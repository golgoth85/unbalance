package core

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		inspect := func(path string, entry os.DirEntry, err error) error {
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
				return fmt.Errorf("Conflitto GATHER file/cartella: %s; sorgente in conflitto preservata", command.Entry)
			}
			for parent := filepath.Dir(target); pathIsInside(paths.DstRoot, parent) && parent != paths.DstRoot; parent = filepath.Dir(parent) {
				if earlier, ok := seen[parent]; ok && !earlier.info.IsDir() {
					return fmt.Errorf("Conflitto GATHER file/cartella: %s; sorgente in conflitto preservata", command.Entry)
				}
				parents[parent] = true
			}
			if earlier, ok := seen[target]; ok {
				equal, err := sameEntryContent(ctx, earlier, current)
				if err != nil {
					return err
				}
				if equal {
					equal, err = sameSourceXattrs(ctx, earlier.path, current.path)
					if err != nil {
						return err
					}
				}
				if !equal {
					return fmt.Errorf("Conflitto GATHER tra sorgenti: %s; sorgente in conflitto preservata", command.Entry)
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
			if equal {
				// Existing attributes may be overwritten/deleted by -X. Adding
				// attributes missing on the destination is a safe repair.
				space, e := changedXattrSpace(target, path)
				if e != nil {
					return e
				}
				equal = space == 0
			}
			if !equal {
				return fmt.Errorf("Conflitto GATHER con destinazione: %s; sorgente in conflitto preservata", command.Entry)
			}
			return nil
		}
		// -R also transfers attributes of the implied parent directories.
		// Inspect them before walking the selected subtree, excluding disk root.
		ancestors := []string{}
		for parent := filepath.Dir(paths.SrcPath); parent != paths.SrcRoot && pathIsInside(paths.SrcRoot, parent); parent = filepath.Dir(parent) {
			ancestors = append(ancestors, parent)
		}
		for i := len(ancestors) - 1; i >= 0; i-- {
			info, e := os.Lstat(ancestors[i])
			if e != nil {
				return e
			}
			if e = inspect(ancestors[i], fs.FileInfoToDirEntry(info), nil); e != nil {
				return e
			}
		}
		err = filepath.WalkDir(paths.SrcPath, inspect)
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
	// Unknown arity is preserved conservatively, never guessed as a switch.
	switches := map[string]bool{}
	for _, name := range strings.Fields("archive recursive relative no-relative implied-dirs no-implied-dirs links copy-links copy-unsafe-links safe-links munge-links copy-dirlinks keep-dirlinks hard-links perms no-perms executability acls xattrs owner group devices specials times omit-dir-times omit-link-times super fake-super sparse preallocate dry-run whole-file checksum one-file-system existing ignore-existing ignore-non-existing update append append-verify inplace partial delay-updates prune-empty-dirs compress compress-old compress-new ipv4 ipv6 numeric-ids human-readable no-human-readable itemize-changes stats progress verbose no-whole-file ignore-times ignore-errors blocking-io no-blocking-io") {
		switches["--"+name] = true
	}

	const shortArguments = "fTBMe@"
	const shortSwitches = "avPRrlptgoDHAisincWSumLkKOJEUNdbzxhICF046X"
	for _, arg := range flags {
		if takesNext {
			out = append(out, arg)
			takesNext = false
			continue
		}
		if arg == "--quiet" || arg == "--msgs2stderr" || arg == "--no-msgs2stderr" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			out = append(out, arg)
			takesNext = !strings.Contains(arg, "=") && !switches[arg]
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			group := arg[1:]
			var cleaned strings.Builder
			for i := 0; i < len(group); i++ {
				option := group[i]
				if option == 'q' {
					continue
				}
				cleaned.WriteByte(option)
				if strings.ContainsRune(shortArguments, rune(option)) {
					cleaned.WriteString(group[i+1:])
					takesNext = i == len(group)-1
					break
				}
				if !strings.ContainsRune(shortSwitches, rune(option)) {
					cleaned.WriteString(group[i+1:])
					takesNext = i == len(group)-1
					break
				}
			}
			if cleaned.Len() > 0 {
				out = append(out, "-"+cleaned.String())
			}
			continue
		}
		out = append(out, arg)
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

func sameSourceXattrs(ctx context.Context, a, b string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	first, err := changedXattrSpace(a, b)
	if err != nil {
		return false, err
	}
	if first != 0 {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	second, err := changedXattrSpace(b, a)
	return second == 0, err
}

func recordPreflightFailure(ctx context.Context, operation *domain.Operation, err error) bool {
	stopped := ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
	status := common.CmdVerificationFailed
	if stopped {
		status = common.CmdStopped
	}
	for _, command := range operation.Commands {
		command.Status = status
		command.Reason = err.Error()
	}
	operation.Finished = time.Now()
	operation.Line = err.Error()
	return stopped
}
