package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
