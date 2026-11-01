package core

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unbalance/daemon/common"
	"unbalance/daemon/domain"
)

func TestGatherBatchProtectsAllVersionsBeforeFirstEffect(t *testing.T) {
	bin := testRsync(t)
	for _, variant := range []string{"different-size", "same-size", "destination", "file-directory", "symlink"} {
		t.Run(variant, func(t *testing.T) {
			a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
			writeResumeFile(t, a, "Film/X/movie.mkv", "version-A")
			writeResumeFile(t, b, "Film/X/movie.mkv", "version-B-long")
			if variant == "same-size" {
				writeResumeFile(t, b, "Film/X/movie.mkv", "version-B")
			}
			if variant == "destination" {
				writeResumeFile(t, b, "Film/X/movie.mkv", "version-A")
				writeResumeFile(t, dst, "Film/X/movie.mkv", "destination-original")
			}
			if variant == "file-directory" {
				if err := os.Remove(filepath.Join(b, "Film/X/movie.mkv")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(b, "Film/X/movie.mkv"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "symlink" {
				for _, root := range []string{a, b} {
					if err := os.Remove(filepath.Join(root, "Film/X/movie.mkv")); err != nil {
						t.Fatal(err)
					}
				}
				os.Symlink("first", filepath.Join(a, "Film/X/movie.mkv"))
				os.Symlink("second", filepath.Join(b, "Film/X/movie.mkv"))
			}
			operation := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "Film/X"}, {Src: b, Dst: dst, Entry: "Film/X"}}}
			called := 0
			// Real effect callback: if the guard is misplaced, this reproduces data loss.
			_, err := runTransferBatch(context.Background(), operation, func(c *domain.Command) error {
				called++
				cmd := exec.Command(bin, "-aRX", "--", c.Entry, c.Dst)
				cmd.Dir = c.Src
				if out, e := cmd.CombinedOutput(); e != nil {
					t.Fatalf("%s: %v", out, e)
				}
				if e := verifyResumeDestination(context.Background(), bin, c); e != nil {
					return e
				}
				_, _, e := removeTransferredSource(c, false)
				return e
			})
			if err == nil || called != 0 {
				t.Fatalf("conflict allowed: calls=%d err=%v", called, err)
			}
			for _, root := range []string{a, b} {
				if _, e := os.Lstat(filepath.Join(root, "Film/X/movie.mkv")); e != nil {
					t.Fatal("source removed", e)
				}
			}
		})
	}
}
func TestGatherBatchMergesDisjointAndIdenticalFiles(t *testing.T) {
	bin := testRsync(t)
	a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
	writeResumeFile(t, a, "Film/X/a", "A")
	writeResumeFile(t, b, "Film/X/b", "B")
	for _, root := range []string{a, b} {
		writeResumeFile(t, root, "Film/X/shared", "same")
	}
	op := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "Film/X"}, {Src: b, Dst: dst, Entry: "Film/X"}}}
	count := 0
	_, err := runTransferBatch(context.Background(), op, func(c *domain.Command) error {
		cmd := exec.Command(bin, "-aRX", "--", c.Entry, c.Dst)
		cmd.Dir = c.Src
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("%s: %v", out, e)
		}
		if e := verifyResumeDestination(context.Background(), bin, c); e != nil {
			return e
		}
		_, _, e := removeTransferredSource(c, false)
		count++
		return e
	})
	if err != nil || count != 2 {
		t.Fatal(count, err)
	}
	for _, name := range []string{"a", "b", "shared"} {
		if _, e := os.Stat(filepath.Join(dst, "Film/X", name)); e != nil {
			t.Fatal(e)
		}
	}
	for _, root := range []string{a, b} {
		if _, e := os.Stat(filepath.Join(root, "Film/X")); !os.IsNotExist(e) {
			t.Fatal("source not removed", e)
		}
	}
}
func TestGatherRechecksDestinationBetweenCommands(t *testing.T) {
	a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
	writeResumeFile(t, a, "Film/X/a", "A")
	writeResumeFile(t, b, "Film/X/b", "B")
	op := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "Film/X"}, {Src: b, Dst: dst, Entry: "Film/X"}}}
	count := 0
	failed, err := runTransferBatch(context.Background(), op, func(c *domain.Command) error {
		count++
		writeResumeFile(t, dst, "Film/X/b", "external version")
		return nil
	})
	if err == nil || count != 1 || failed != op.Commands[1] {
		t.Fatal(count, failed, err)
	}
}
func TestScatterOverlapIgnoresSplitAndEmptyDirectories(t *testing.T) {
	src, other := t.TempDir(), t.TempDir()
	writeResumeFile(t, src, "Serie/Show/Season 2/episode", "content")
	writeResumeFile(t, other, "Serie/Show/Season 1/episode", "unrelated")
	for _, path := range []string{"Serie/Show", "Serie/Show/Season 2"} {
		overlap, err := transferHasOverlap(context.Background(), filepath.Join(src, path), filepath.Join(other, path))
		if err != nil || overlap {
			t.Fatal(path, overlap, err)
		}
	}
	os.MkdirAll(filepath.Join(other, "Serie/Show/Season 2"), 0755)
	overlap, err := transferHasOverlap(context.Background(), filepath.Join(src, "Serie/Show"), filepath.Join(other, "Serie/Show"))
	if err != nil || overlap {
		t.Fatal(overlap, err)
	}
	writeResumeFile(t, other, "Serie/Show/Season 2/episode", "truncated")
	overlap, err = transferHasOverlap(context.Background(), filepath.Join(src, "Serie/Show"), filepath.Join(other, "Serie/Show"))
	if err != nil || !overlap {
		t.Fatal(overlap, err)
	}
}
func TestFilterArgumentsKeepLetterQ(t *testing.T) {
	bin := testRsync(t)
	src, dst := t.TempDir(), t.TempDir()
	writeResumeFile(t, src, "share/sub/image.qcow2", strings.Repeat("x", 900000))
	flags := []string{"-f+ */", "-f+ *.qcow2", "-f- *"}
	if !reflect.DeepEqual(flags, comparisonFlags(flags)) {
		t.Fatal(comparisonFlags(flags))
	}
	size, _, err := estimateTransfer(context.Background(), bin, &domain.Command{Src: src, Dst: dst, Entry: "share"}, flags)
	if err != nil || size != 900000 {
		t.Fatal(size, err)
	}
	separate := []string{"-f", "+ *.qcow2", "--include", "-q*", "-q", "-aqX"}
	if !reflect.DeepEqual(comparisonFlags(separate), []string{"-f", "+ *.qcow2", "--include", "-q*", "-aX"}) {
		t.Fatal(comparisonFlags(separate))
	}
}
func TestStopBetweenAdmissionAndWorkerCannotBeLost(t *testing.T) {
	c := &Core{}
	c.armOperation() // synchronous admission
	c.stopped.Store(true)
	c.cancelPlanContext() // mailbox handles STOP before worker
	called := false
	_, err := runTransferBatch(c.currentComparisonContext(), &domain.Operation{OpKind: common.OpGatherMove}, func(*domain.Command) error { called = true; return nil })
	if err == nil || called || !c.stopped.Load() {
		t.Fatal("STOP lost", err)
	}
	c.clearPlanContext()
	c.armOperation()
	if c.stopped.Load() || c.currentComparisonContext().Err() != nil {
		t.Fatal("next operation remains cancelled")
	}
	c.clearPlanContext()
}

func TestGatherXattrVersionsCannotOverwriteEachOther(t *testing.T) {
	for _, targetConflict := range []bool{false, true} {
		a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
		writeResumeFile(t, a, "share/file", "same bytes")
		if err := unix.Setxattr(filepath.Join(a, "share/file"), "user.synthetic", []byte("version A"), 0); err != nil {
			t.Fatal(err)
		}
		second := b
		if targetConflict {
			second = dst
		}
		writeResumeFile(t, second, "share/file", "same bytes")
		if err := unix.Setxattr(filepath.Join(second, "share/file"), "user.synthetic", []byte("version B"), 0); err != nil {
			t.Fatal(err)
		}
		commands := []*domain.Command{{Src: a, Dst: dst, Entry: "share"}}
		if !targetConflict {
			commands = append(commands, &domain.Command{Src: b, Dst: dst, Entry: "share"})
		}
		called := false
		_, err := runTransferBatch(context.Background(), &domain.Operation{OpKind: common.OpGatherMove, Commands: commands}, func(*domain.Command) error { called = true; return nil })
		if err == nil || called {
			t.Fatal("xattr conflict accepted", targetConflict, err)
		}
		for _, root := range []string{a, second} {
			if _, err := os.Stat(filepath.Join(root, "share/file")); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestGatherRepairsMissingXattrsAndMergesIdenticalVersions(t *testing.T) {
	bin := testRsync(t)
	a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
	for _, root := range []string{a, b, dst} {
		writeResumeFile(t, root, "share/file", "same bytes")
	}
	for _, root := range []string{a, b} {
		if err := unix.Setxattr(filepath.Join(root, "share/file"), "user.synthetic", []byte("same metadata"), 0); err != nil {
			t.Fatal(err)
		}
	}
	op := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "share"}, {Src: b, Dst: dst, Entry: "share"}}}
	_, err := runTransferBatch(context.Background(), op, func(c *domain.Command) error {
		cmd := exec.Command(bin, "-aRX", "--", c.Entry, c.Dst)
		cmd.Dir = c.Src
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("%s: %v", out, e)
		}
		if e := verifyResumeDestination(context.Background(), bin, c); e != nil {
			return e
		}
		_, _, e := removeTransferredSource(c, false)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	bytes := make([]byte, 100)
	n, err := unix.Getxattr(filepath.Join(dst, "share/file"), "user.synthetic", bytes)
	if err != nil || string(bytes[:n]) != "same metadata" {
		t.Fatal(n, err)
	}
	for _, root := range []string{a, b} {
		if _, err := os.Stat(filepath.Join(root, "share/file")); !os.IsNotExist(err) {
			t.Fatal("source remains", err)
		}
	}
}

func TestGatherDirectoryXattrConflictsProtectSelectedAndImpliedDirectories(t *testing.T) {
	bin := testRsync(t)
	for _, dir := range []string{"Film/X", "Film/X/sub", "Film"} {
		for _, againstDestination := range []bool{false, true} {
			t.Run(dir+fmt.Sprint(againstDestination), func(t *testing.T) {
				a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
				writeResumeFile(t, a, "Film/X/sub/a", "A")
				writeResumeFile(t, b, "Film/X/sub/b", "B")
				other := b
				if againstDestination {
					other = dst
					os.MkdirAll(filepath.Join(dst, dir), 0755)
				}
				if !againstDestination {
					if err := unix.Setxattr(filepath.Join(a, dir), "user.tag", []byte("version-A"), 0); err != nil {
						t.Fatal(err)
					}
				}
				if err := unix.Setxattr(filepath.Join(other, dir), "user.tag", []byte("version-B"), 0); err != nil {
					t.Fatal(err)
				}
				commands := []*domain.Command{{Src: a, Dst: dst, Entry: "Film/X"}}
				if !againstDestination {
					commands = append(commands, &domain.Command{Src: b, Dst: dst, Entry: "Film/X"})
				}
				effects := 0
				_, err := runTransferBatch(context.Background(), &domain.Operation{OpKind: common.OpGatherMove, Commands: commands}, func(c *domain.Command) error {
					effects++
					cmd := exec.Command(bin, "-avPR", "-X", "--", c.Entry, c.Dst)
					cmd.Dir = c.Src
					if out, e := cmd.CombinedOutput(); e != nil {
						t.Fatalf("%s: %v", out, e)
					}
					if e := verifyResumeDestination(context.Background(), bin, c); e != nil {
						return e
					}
					_, _, e := removeTransferredSource(c, true)
					return e
				})
				if err == nil || effects != 0 {
					t.Fatal("directory xattr conflict not blocked", effects, err)
				}
				for _, source := range commands {
					if _, e := os.Stat(filepath.Join(source.Src, source.Entry)); e != nil {
						t.Fatal("source removed", e)
					}
				}
				value := make([]byte, 100)
				n, e := unix.Getxattr(filepath.Join(other, dir), "user.tag", value)
				if e != nil || string(value[:n]) != "version-B" {
					t.Fatal("attribute lost", n, e)
				}
			})
		}
	}
}
func TestGatherCompletesMissingDirectoryXattrsAndDocumentsLastMetadata(t *testing.T) {
	bin := testRsync(t)
	a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
	for _, root := range []string{a, b} {
		writeResumeFile(t, root, "Film/X/sub/identical", "same bytes")
		for _, dir := range []string{"Film", "Film/X", "Film/X/sub"} {
			if err := unix.Setxattr(filepath.Join(root, dir), "user.tag", []byte("same metadata"), 0); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Chmod(filepath.Join(a, "Film/X/sub/identical"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(b, "Film/X/sub/identical"), 0600); err != nil {
		t.Fatal(err)
	}
	first, last := time.Unix(1600000000, 0), time.Unix(1500000000, 0)
	os.Chtimes(filepath.Join(a, "Film/X/sub/identical"), first, first)
	os.Chtimes(filepath.Join(b, "Film/X/sub/identical"), last, last)
	op := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "Film/X"}, {Src: b, Dst: dst, Entry: "Film/X"}}}
	_, err := runTransferBatch(context.Background(), op, func(c *domain.Command) error {
		cmd := exec.Command(bin, "-avPR", "-X", "--", c.Entry, c.Dst)
		cmd.Dir = c.Src
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("%s: %v", out, e)
		}
		if e := verifyResumeDestination(context.Background(), bin, c); e != nil {
			return e
		}
		_, _, e := removeTransferredSource(c, false)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"Film", "Film/X", "Film/X/sub"} {
		value := make([]byte, 100)
		n, e := unix.Getxattr(filepath.Join(dst, dir), "user.tag", value)
		if e != nil || string(value[:n]) != "same metadata" {
			t.Fatal(dir, n, e)
		}
	}
	info, err := os.Stat(filepath.Join(dst, "Film/X/sub/identical"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 || info.ModTime().Unix() != last.Unix() {
		t.Fatal("last rsync metadata policy changed", info)
	}
}
func TestGatherVolatileAttributesRemainExplicitConflicts(t *testing.T) {
	a, b, dst := t.TempDir(), t.TempDir(), t.TempDir()
	for i, root := range []string{a, b} {
		writeResumeFile(t, root, "share/file", "same")
		if err := unix.Setxattr(filepath.Join(root, "share/file"), "user.DOSATTRIB", []byte(fmt.Sprint(i)), 0); err != nil {
			t.Fatal(err)
		}
	}
	effects := 0
	_, err := runTransferBatch(context.Background(), &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{Src: a, Dst: dst, Entry: "share"}, {Src: b, Dst: dst, Entry: "share"}}}, func(*domain.Command) error { effects++; return nil })
	if err == nil || effects != 0 {
		t.Fatal("volatile xattrs silently ignored", effects, err)
	}
}
func TestPreflightStopIsRecordedAsStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	operation := &domain.Operation{OpKind: common.OpGatherMove, Commands: []*domain.Command{{}, {}}}
	failed, err := runTransferBatch(ctx, operation, func(*domain.Command) error { t.Fatal("effect after STOP"); return nil })
	if failed != nil || err == nil {
		t.Fatal(failed, err)
	}
	if !recordPreflightFailure(ctx, operation, err) {
		t.Fatal("not classified as STOP")
	}
	for _, command := range operation.Commands {
		if command.Status != common.CmdStopped {
			t.Fatal(command.Status)
		}
	}
	if recordPreflightFailure(context.Background(), operation, fmt.Errorf("conflict")) {
		t.Fatal("conflict treated as STOP")
	}
	for _, command := range operation.Commands {
		if command.Status != common.CmdVerificationFailed {
			t.Fatal(command.Status)
		}
	}
}
func TestGroupedQuietAndUnknownValueAreNotCorrupted(t *testing.T) {
	bin := testRsync(t)
	src, dst := t.TempDir(), t.TempDir()
	writeResumeFile(t, src, "share/file", "1234567")
	flags := []string{"-qf", "- *.x"}
	if !reflect.DeepEqual(comparisonFlags(flags), []string{"-f", "- *.x"}) {
		t.Fatal(comparisonFlags(flags))
	}
	size, _, err := estimateTransfer(context.Background(), bin, &domain.Command{Src: src, Dst: dst, Entry: "share"}, flags)
	if err != nil || size != 7 {
		t.Fatal(size, err)
	}
	for _, valueOption := range []string{"--compare-dest", "--copy-dest", "--link-dest", "--future-value-option"} {
		flags := []string{valueOption, "-q"}
		if !reflect.DeepEqual(flags, comparisonFlags(flags)) {
			t.Fatal(comparisonFlags(flags))
		}
	}
	if err := os.Mkdir(filepath.Join(dst, "-q"), 0755); err != nil {
		t.Fatal(err)
	}
	size, _, err = estimateTransfer(context.Background(), bin, &domain.Command{Src: src, Dst: dst, Entry: "share"}, []string{"--compare-dest", "-q"})
	if err != nil || size != 7 {
		t.Fatal(size, err)
	}
}

func TestCommonSwitchesBeforeQuietProduceRealStatistics(t *testing.T) {
	bin := testRsync(t)
	src, dst := t.TempDir(), t.TempDir()
	writeResumeFile(t, src, "share/file", "1234567")
	for _, flags := range [][]string{
		{"--size-only", "-q"}, {"--no-motd", "-q"}, {"--mkpath", "-q"},
		{"--force", "-q"}, {"--8-bit-output", "-q"}, {"-vy", "-q"}, {"-yq"}, {"-8q"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			size, _, err := estimateTransfer(context.Background(), bin, &domain.Command{Src: src, Dst: dst, Entry: "share"}, flags)
			if err != nil || size != 7 {
				t.Fatal(size, err, comparisonFlags(flags))
			}
		})
	}
}
