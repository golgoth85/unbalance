package core

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"unbalance/daemon/common"
	"unbalance/daemon/domain"
)

func testRsync(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("TEST_RSYNC_BIN")
	if bin == "" {
		var err error
		bin, err = exec.LookPath("rsync")
		if err != nil {
			t.Skip("rsync non disponibile")
		}
	}
	return bin
}
func writeResumeFile(t *testing.T, root, name, value string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(value), 0644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1700000000, 0)
	if err := os.Chtimes(p, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}
func TestResumeComparisonAndVerification(t *testing.T) {
	bin := testRsync(t)
	ctx := context.Background()
	src := t.TempDir()
	dst := t.TempDir()
	// Equal files, missing file and existing different file. Names must remain
	// argv data, including shell metacharacters and embedded newlines.
	name := "share/Serie ' ; $(echo NO)\nnuova"
	writeResumeFile(t, src, name+"/equal.cbz", "equal")
	writeResumeFile(t, dst, name+"/equal.cbz", "equal")
	writeResumeFile(t, src, name+"/missing.cbz", "missing")
	writeResumeFile(t, src, name+"/different.cbz", "new data")
	writeResumeFile(t, dst, name+"/different.cbz", "old")
	command := &domain.Command{Src: src, Dst: dst, Entry: name}
	size, space, err := estimateTransfer(ctx, bin, command, []string{"-X"})
	if err != nil {
		t.Fatal(err)
	}
	if size != 15 || space <= size {
		t.Fatalf("size=%d space=%d", size, space)
	}
	if _, err := os.Stat(filepath.Join(dst, name, "missing.cbz")); !os.IsNotExist(err) {
		t.Fatal("dry-run ha scritto")
	}
	if err := verifyResumeDestination(ctx, bin, command); err == nil {
		t.Fatal("verifica ha ignorato differenze")
	}
	// A real isolated copy is followed by a second comparison and checksum.
	cmd := exec.Command(bin, "-aRX", "--", name, dst)
	cmd.Dir = src
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", output, err)
	}
	size, space, err = estimateTransfer(ctx, bin, command, nil)
	if err != nil || size != 0 || space != 0 {
		t.Fatalf("dopo copia: %d %d %v", size, space, err)
	}
	if err := verifyResumeDestination(ctx, bin, command); err != nil {
		t.Fatal(err)
	}
	// Same size and mtime hides a content conflict from quick-check, but not
	// from the mandatory checksum verification before MOVE source deletion.
	writeResumeFile(t, dst, name+"/equal.cbz", "other")
	size, _, err = estimateTransfer(ctx, bin, command, nil)
	if err != nil || size != 0 {
		t.Fatalf("quick-check: %d %v", size, err)
	}
	if err := verifyResumeDestination(ctx, bin, command); err == nil {
		t.Fatal("checksum non ha rilevato conflitto")
	}
	size, _, err = estimateTransfer(ctx, bin, command, []string{"--checksum"})
	if err != nil || size != 5 {
		t.Fatalf("checksum estimate: %d %v", size, err)
	}
	// User skip/filter flags are honored for planning, never for source deletion.
	size, _, err = estimateTransfer(ctx, bin, command, []string{"--checksum", "--ignore-existing"})
	if err != nil || size != 0 {
		t.Fatalf("ignore-existing: %d %v", size, err)
	}
	if err := verifyResumeDestination(ctx, bin, command); err == nil {
		t.Fatal("skip flags devono preservare sorgente")
	}
}
func TestResumeErrorsAndCancellation(t *testing.T) {
	bin := testRsync(t)
	src := t.TempDir()
	dst := t.TempDir()
	writeResumeFile(t, src, "share/file", "data")
	c := &domain.Command{Src: src, Dst: dst, Entry: "share"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := estimateTransfer(ctx, bin, c, nil); err == nil {
		t.Fatal("cancellazione ignorata")
	}
	if _, _, err := estimateTransfer(context.Background(), bin, c, []string{"--not-a-real-flag"}); err == nil {
		t.Fatal("errore rsync ignorato")
	}
	if _, _, err := estimateTransfer(context.Background(), bin, c, []string{"--log-file=" + filepath.Join(dst, "log")}); err == nil {
		t.Fatal("scrittura laterale consentita")
	}
	c.Entry = "../escape"
	if _, _, err := estimateTransfer(context.Background(), bin, c, nil); err == nil {
		t.Fatal("traversal consentito")
	}
	c.Entry = "share"
	if err := os.Symlink(t.TempDir(), filepath.Join(dst, "share")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := estimateTransfer(context.Background(), bin, c, nil); err == nil {
		t.Fatal("symlink fuori root consentito")
	}
}
func TestTransferStatsFailClosed(t *testing.T) {
	for _, s := range []string{"", "Total transferred file size: 0 bytes", "Total transferred file size: rubbish\nNumber of created files: 0\nNumber of regular files transferred: 0", "Total transferred file size: 18446744073709551615 bytes\nNumber of created files: 1\nNumber of regular files transferred: 1"} {
		if _, _, err := parseTransferStats(s); err == nil {
			t.Fatalf("accettato %q", s)
		}
	}
	text := "Number of created files: 1,001 (reg: 1,000, dir: 1)\nNumber of regular files transferred: 1,000\nTotal transferred file size: 123,456 bytes\n"
	size, _, err := parseTransferStats(text)
	if err != nil || size != 123456 {
		t.Fatal(size, err)
	}
}

func TestScatterCommandsUseDeltaAndKeepCompleteFolders(t *testing.T) {
	c := &Core{ctx: &domain.Context{Config: domain.Config{DryRun: true, RsyncArgs: []string{"-X"}}}, state: &domain.State{Status: common.OpScatterCopy, Unraid: &domain.Unraid{Disks: []*domain.Disk{{Path: "/target"}}}}}
	plan := domain.Plan{BytesToTransfer: 600, VDisks: map[string]*domain.VDisk{"/target": {Bin: &domain.Bin{Items: []*domain.Item{
		{Path: "share/complete", Location: "/source", Size: 3000, Estimated: true, TransferSize: 0},
		{Path: "share/partial", Location: "/source", Size: 9000, Estimated: true, TransferSize: 600},
	}}}}}
	operation := c.createScatterOperation(plan)
	if len(operation.Commands) != 2 || operation.BytesToTransfer != 600 || operation.Commands[0].Size != 0 || operation.Commands[1].Size != 600 {
		t.Fatalf("unexpected operation: %+v", operation)
	}
	restored, err := cloneOperation(*operation)
	if err != nil || !restored.Commands[0].Estimated {
		t.Fatal("history lost verification requirement", err)
	}
}
func TestStopCancelsComparisonContext(t *testing.T) {
	c := &Core{}
	ctx := c.newPlanContext()
	if c.currentComparisonContext() != ctx {
		t.Fatal("comparison context not registered")
	}
	c.cancelPlanContext()
	if c.currentComparisonContext().Err() == nil {
		t.Fatal("STOP did not cancel comparison")
	}
	c.clearPlanContext()
	if c.currentComparisonContext().Err() != nil {
		t.Fatal("cancelled context leaked into next operation")
	}
}

func TestResumeChoosesPartialDestinationBeforeFreeSpace(t *testing.T) {
	bin := testRsync(t)
	src := t.TempDir()
	a := t.TempDir()
	b := t.TempDir()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("share/series/%02d.cbz", i)
		writeResumeFile(t, src, name, strings.Repeat("x", 1000))
		if i < 9 {
			writeResumeFile(t, b, name, strings.Repeat("x", 1000))
		}
	}
	disks := []*domain.Disk{{Name: "A", Path: a, Free: 1e9}, {Name: "B", Path: b, Free: 1e8}}
	item := &domain.Item{Name: "series", Path: "share/series", Location: src, Size: 10000}
	estimate := func(ctx context.Context, c *domain.Command) (uint64, uint64, error) {
		return estimateTransfer(ctx, bin, c, []string{"-X"})
	}
	bins, err := resumeBins(context.Background(), disks, []*domain.Item{item}, func(*domain.Disk) uint64 { return 100 }, estimate, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if bins[a] != nil || bins[b] == nil || bins[b].Size != 1000 {
		t.Fatalf("wrong allocation: %+v", bins)
	}
	c := &Core{state: &domain.State{Unraid: &domain.Unraid{Disks: append(disks, &domain.Disk{Path: src})}}}
	command := &domain.Command{Src: src, Dst: b, Entry: item.Path}
	if err := c.otherDestinationCopies(command); err != nil {
		t.Fatal(err)
	}
	writeResumeFile(t, a, "share/series/00.cbz", "truncated")
	if err := c.otherDestinationCopies(command); err == nil {
		t.Fatal("MOVE would leave another partial copy")
	}
}
func TestQuietAndStderrFlagsDoNotBreakComparison(t *testing.T) {
	bin := testRsync(t)
	src := t.TempDir()
	dst := t.TempDir()
	writeResumeFile(t, src, "share/file", "content")
	for _, flags := range [][]string{{"-q"}, {"--quiet"}, {"--msgs2stderr"}, {"-aqX"}} {
		n, _, err := estimateTransfer(context.Background(), bin, &domain.Command{Src: src, Dst: dst, Entry: "share"}, flags)
		if err != nil || n != 7 {
			t.Fatalf("%v: %d %v", flags, n, err)
		}
	}
}
func TestXattrSpaceIncludesAllocationAndMetadataOnlyChanges(t *testing.T) {
	bin := testRsync(t)
	src := t.TempDir()
	dst := t.TempDir()
	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("share/%d", i)
		writeResumeFile(t, src, name, "a")
		if err := unix.Setxattr(filepath.Join(src, name), "user.synthetic", []byte(strings.Repeat("x", 60000)), 0); err != nil {
			t.Fatal(err)
		}
	}
	command := &domain.Command{Src: src, Dst: dst, Entry: "share"}
	_, space, err := estimateTransfer(context.Background(), bin, command, []string{"-X"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-aRX", "--", "share", dst)
	cmd.Dir = src
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	var allocated uint64
	filepath.Walk(filepath.Join(dst, "share"), func(_ string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		allocated += uint64(info.Sys().(*syscall.Stat_t).Blocks) * 512
		return nil
	})
	if space < allocated {
		t.Fatalf("xattr underestimate: %d < %d", space, allocated)
	}
	n, required, err := estimateTransfer(context.Background(), bin, command, []string{"-X"})
	if err != nil || n != 0 || required != 0 {
		t.Fatalf("complete copy: %d %d %v", n, required, err)
	}
	if err := unix.Setxattr(filepath.Join(src, "share/0"), "user.synthetic", []byte(strings.Repeat("y", 60000)), 0); err != nil {
		t.Fatal(err)
	}
	n, required, err = estimateTransfer(context.Background(), bin, command, []string{"-X"})
	if err != nil || n != 0 || required < 60000 {
		t.Fatalf("metadata-only xattr: %d %d %v", n, required, err)
	}
}
func TestAllocationErrorsInvalidateWholePlan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := resumeBins(ctx, []*domain.Disk{{Path: "/target"}}, []*domain.Item{{Path: "share"}}, func(*domain.Disk) uint64 { return 0 }, func(context.Context, *domain.Command) (uint64, uint64, error) { return 0, 0, nil }, func(string) {})
	if err == nil {
		t.Fatal("cancel ignored")
	}
	sentinel := errors.New("comparison failure")
	_, err = resumeBins(context.Background(), []*domain.Disk{{Path: "/target"}}, []*domain.Item{{Path: "share"}}, func(*domain.Disk) uint64 { return 0 }, func(context.Context, *domain.Command) (uint64, uint64, error) { return 0, 0, sentinel }, func(string) {})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if common.CmdVerificationFailed == common.CmdFlagged || common.CmdVerificationFailed == common.CmdSourceRemoval {
		t.Fatal("rollback exposes source deletion")
	}
}

func TestCompleteFoldersFitBelowReserve(t *testing.T) {
	disks := []*domain.Disk{{Path: "/target", Free: 50}}
	items := []*domain.Item{{Path: "share/one"}, {Path: "share/two"}, {Path: "share/missing"}}
	bins, err := resumeBins(context.Background(), disks, items, func(*domain.Disk) uint64 { return 100 }, func(_ context.Context, c *domain.Command) (uint64, uint64, error) {
		if strings.HasSuffix(c.Entry, "missing") {
			return 1, 4096, nil
		}
		return 0, 0, nil
	}, func(string) {})
	if err != nil || bins["/target"] == nil || len(bins["/target"].Items) != 2 {
		t.Fatal(bins, err)
	}
}

func TestVerifyDestinationQuickWithPreexistingChecksum(t *testing.T) {
	bin := testRsync(t)
	ctx := context.Background()
	src := t.TempDir()
	dst := t.TempDir()
	name := "share/Serie ' ; $(echo NO)\nnuova"
	command := &domain.Command{Src: src, Dst: dst, Entry: name}
	writeResumeFile(t, src, name+"/new.mkv", "brand new")
	writeResumeFile(t, src, name+"/same.mkv", "aaaa")
	writeResumeFile(t, src, name+"/sub dir/stale\nfile.mkv", "good")

	// Nothing on the destination yet: empty, non-nil snapshot.
	snap, err := snapshotDestination(ctx, command)
	if err != nil || snap == nil || len(snap) != 0 {
		t.Fatalf("snapshot vuoto: %v %v", snap, err)
	}

	// Pre-existing copies: one identical, one with same size and mtime but
	// different content (hidden from quick-check), one only on destination.
	writeResumeFile(t, dst, name+"/same.mkv", "aaaa")
	writeResumeFile(t, dst, name+"/sub dir/stale\nfile.mkv", "evil")
	writeResumeFile(t, dst, name+"/only-dst.nfo", "x")
	snap, err = snapshotDestination(ctx, command)
	if err != nil || len(snap) != 3 {
		t.Fatalf("snapshot: %q %v", snap, err)
	}
	cmd := exec.Command(bin, "-aRX", "--", name, dst)
	cmd.Dir = src
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", output, err)
	}
	if err := verifyDestination(ctx, bin, command, snap, false); err == nil {
		t.Fatal("checksum dei file preesistenti non ha rilevato il conflitto")
	}
	if err := verifyDestination(ctx, bin, command, nil, false); err == nil {
		t.Fatal("snapshot assente deve usare il checksum completo")
	}
	if err := verifyDestination(ctx, bin, command, []string{}, true); err == nil {
		t.Fatal("verifica completa non ha rilevato il conflitto")
	}
	// Without the stale file in the snapshot the quick check alone passes:
	// that is the documented trust in rsync's transfer checksum.
	if err := verifyDestination(ctx, bin, command, []string{filepath.Join(name, "same.mkv")}, false); err != nil {
		t.Fatalf("controllo rapido: %v", err)
	}

	// Fix the stale copy: quick + targeted checksum pass, destination-only
	// files are ignored, full check agrees.
	writeResumeFile(t, dst, name+"/sub dir/stale\nfile.mkv", "good")
	if err := verifyDestination(ctx, bin, command, snap, false); err != nil {
		t.Fatalf("verifica rapida dopo correzione: %v", err)
	}
	if err := verifyResumeDestination(ctx, bin, command); err != nil {
		t.Fatalf("verifica completa dopo correzione: %v", err)
	}

	// Missing file (e.g. excluded by a user filter) and later source change are
	// caught by the quick check even with an empty snapshot.
	if err := os.Remove(filepath.Join(dst, name, "new.mkv")); err != nil {
		t.Fatal(err)
	}
	if err := verifyDestination(ctx, bin, command, []string{}, false); err == nil {
		t.Fatal("file mancante non rilevato")
	}
	writeResumeFile(t, dst, name+"/new.mkv", "brand new")
	writeResumeFile(t, src, name+"/new.mkv", "brand new, edited")
	if err := verifyDestination(ctx, bin, command, []string{}, false); err == nil {
		t.Fatal("sorgente modificata non rilevata")
	}

	// STOP cancels both the snapshot walk and the verification.
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := snapshotDestination(cancelled, command); err == nil {
		t.Fatal("snapshot ha ignorato la cancellazione")
	}
	if err := verifyDestination(cancelled, bin, command, snap, false); err == nil {
		t.Fatal("verifica ha ignorato la cancellazione")
	}
}

func TestSnapshotDestinationRejectsUnsafePaths(t *testing.T) {
	dst := t.TempDir()
	outside := t.TempDir()
	writeResumeFile(t, outside, "secret/file", "x")
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dst, "share")); err != nil {
		t.Fatal(err)
	}
	c := &domain.Command{Src: t.TempDir(), Dst: dst, Entry: "share"}
	if _, err := snapshotDestination(context.Background(), c); err == nil {
		t.Fatal("symlink fuori root consentito")
	}
	c.Entry = "../escape"
	if _, err := snapshotDestination(context.Background(), c); err == nil {
		t.Fatal("traversal consentito")
	}
}
