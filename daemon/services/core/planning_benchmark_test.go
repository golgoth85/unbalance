package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unbalance/daemon/domain"
)

// Small reproducible metadata workload; no estimate of real library duration.
func BenchmarkResumeDestinationCount(b *testing.B) {
	binary := os.Getenv("TEST_RSYNC_BIN")
	if binary == "" {
		var err error
		binary, err = exec.LookPath("rsync")
		if err != nil {
			b.Skip("rsync unavailable")
		}
	}
	for _, count := range []int{2, 4} {
		b.Run(fmt.Sprintf("%d-disks", count), func(b *testing.B) {
			src := b.TempDir()
			items := []*domain.Item{}
			for i := 0; i < 3; i++ {
				entry := fmt.Sprintf("share/series-%d", i)
				os.MkdirAll(filepath.Join(src, entry), 0755)
				for j := 0; j < 5; j++ {
					if err := os.WriteFile(filepath.Join(src, entry, fmt.Sprint(j)), make([]byte, 512), 0644); err != nil {
						b.Fatal(err)
					}
				}
				items = append(items, &domain.Item{Path: entry, Location: src, Size: 2560})
			}
			disks := []*domain.Disk{}
			for i := 0; i < count; i++ {
				disks = append(disks, &domain.Disk{Path: b.TempDir(), Free: 1e9})
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_, err := resumeBins(context.Background(), disks, items, func(*domain.Disk) uint64 { return 4096 }, func(ctx context.Context, c *domain.Command) (uint64, uint64, error) {
					return estimateTransfer(ctx, binary, c, []string{"-X"})
				}, func(string) {})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
