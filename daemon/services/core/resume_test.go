package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func TestResumeBinAllocatesDeltaAndAllZeroItems(t *testing.T) {
	ctx := context.Background()
	items := []*domain.Item{{Name: "existing", Path: "share/existing", Location: "/source", Size: 3000}, {Name: "missing", Path: "share/missing", Location: "/source", Size: 600}, {Name: "also-existing", Path: "share/also-existing", Location: "/source", Size: 500}}
	estimate := func(_ context.Context, c *domain.Command) (uint64, uint64, error) {
		if c.Entry == "share/missing" {
			return 600, 650, nil
		}
		return 0, 0, nil
	}
	disk := &domain.Disk{Name: "disk3", Path: "/target", Free: 1000}
	bin, err := resumeBin(ctx, disk, items, 100, estimate, func(string) {})
	if err != nil || len(bin.Items) != 3 || bin.Size != 600 || bin.RequiredSpace != 650 {
		t.Fatalf("bin=%+v err=%v", bin, err)
	}
	for _, item := range items {
		if item.Estimated {
			t.Fatal("modificato item originale")
		}
	}
	disk.Free = 50
	bin, err = resumeBin(ctx, disk, items, 100, estimate, func(string) {})
	if err != nil || len(bin.Items) != 2 || bin.Size != 0 {
		t.Fatalf("zero sotto riserva: %+v %v", bin, err)
	}
	remaining := removeItems(append([]*domain.Item{}, items...), bin.Items)
	if len(remaining) != 1 || remaining[0].Name != "missing" {
		t.Fatal("assegnazione ha perso item")
	}
	// A second destination gets its own estimate, not the first disk's delta.
	disk.Path = "/target2"
	disk.Free = 1000
	estimate2 := func(_ context.Context, c *domain.Command) (uint64, uint64, error) {
		if c.Dst != "/target2" {
			t.Fatal(c.Dst)
		}
		return 600, 650, nil
	}
	bin, err = resumeBin(ctx, disk, remaining, 100, estimate2, func(string) {})
	if err != nil || len(bin.Items) != 1 {
		t.Fatal(bin, err)
	}
	_, err = resumeBin(ctx, disk, items, 100, func(context.Context, *domain.Command) (uint64, uint64, error) { return 0, 0, fmt.Errorf("read failed") }, func(string) {})
	if err == nil {
		t.Fatal("errore non ha invalidato bin")
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
