package core

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"unbalance/daemon/common"
	"unbalance/daemon/domain"
)

// estimateTransfer uses the same selection/quick-check rules as the transfer.
// Changed files reserve their full replacement size: rsync creates temporary
// files by default, so subtracting the old destination size would be unsafe.
func estimateTransfer(ctx context.Context, binary string, command *domain.Command, flags []string) (uint64, uint64, error) {
	paths, err := safeCommandPaths(command)
	if err != nil {
		return 0, 0, err
	}
	if err = validateExistingPathBoundary(paths.SrcRoot, paths.SrcPath); err != nil {
		return 0, 0, err
	}
	if err = validateCreatablePathBoundary(paths.DstRoot, paths.DstPath); err != nil {
		return 0, 0, err
	}
	if err = validateRsyncArgs(flags); err != nil {
		return 0, 0, err
	}
	args := append([]string{common.RsyncArgs}, comparisonFlags(flags)...)
	args = append(args, "--dry-run", "--stats", "--no-human-readable", "--out-format=", "--info=all0,stats2", "--", paths.Entry, paths.DstRoot)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = paths.SrcRoot
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	// With per-file output disabled, stdout only contains the small stats block.
	var stdout limitedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &limitedOutput{}
	if err := cmd.Run(); err != nil {
		return 0, 0, fmt.Errorf("rsync comparison failed: %w", err)
	}
	size, required, err := parseTransferStats(stdout.String())
	if err != nil {
		return 0, 0, err
	}
	extra, err := allocationExtra(ctx, paths.SrcPath, paths.DstPath, copiesXattrs(flags))
	if err != nil {
		return 0, 0, err
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(paths.DstRoot, &fs); err != nil {
		return 0, 0, err
	}
	block := uint64(fs.Frsize)
	if block == 0 {
		block = uint64(fs.Bsize)
	}
	if block > 4096 {
		entries := (required - size) / 4096
		if entries > (math.MaxUint64-required)/(block-4096) {
			return 0, 0, fmt.Errorf("space overflow")
		}
		required += entries * (block - 4096)
	}
	if extra > math.MaxUint64-required {
		return 0, 0, fmt.Errorf("space overflow")
	}
	return size, required + extra, nil
}

// Do not retain arbitrarily large output or filenames from a failed process.
type limitedOutput struct{ bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if left := 65536 - b.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}

func parseTransferStats(output string) (uint64, uint64, error) {
	var size, created, transferred uint64
	found := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if key != "Total transferred file size" && key != "Number of created files" && key != "Number of regular files transferred" {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			return 0, 0, fmt.Errorf("missing rsync statistic: %s", key)
		}
		n, err := strconv.ParseUint(strings.ReplaceAll(fields[0], ",", ""), 10, 64)
		if err != nil || found[key] {
			return 0, 0, fmt.Errorf("invalid rsync statistic: %s", key)
		}
		found[key] = true
		switch key {
		case "Total transferred file size":
			size = n
		case "Number of created files":
			created = n
		case "Number of regular files transferred":
			transferred = n
		}
	}
	if len(found) != 3 {
		return 0, 0, fmt.Errorf("rsync did not return complete transfer statistics")
	}
	// One block per transferred/created entry covers alignment and directory
	// overhead conservatively; the configured disk reserve remains additional.
	if created > math.MaxUint64-transferred || created+transferred > (math.MaxUint64-size)/4096 {
		return 0, 0, fmt.Errorf("rsync space estimate overflow")
	}
	return size, size + (created+transferred)*4096, nil
}

type transferEstimator func(context.Context, *domain.Command) (uint64, uint64, error)

// This check deliberately ignores user filters and skip flags. A successful
// filtered copy is not proof that it is safe to delete the whole source tree.
func verifyResumeDestination(ctx context.Context, binary string, command *domain.Command) error {
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
	cmd := exec.CommandContext(ctx, binary, "-aRXnc", "--itemize-changes", "--out-format=%i", "--", paths.Entry, paths.DstRoot)
	cmd.Dir = paths.SrcRoot
	var stdout limitedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &limitedOutput{}
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("destination verification failed: %w", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		return fmt.Errorf("destination differs from source; source preserved")
	}
	return nil
}

// Only presentation flags are removed; selection/comparison semantics stay intact.
func comparisonFlags(flags []string) []string {
	out := []string{}
	for _, f := range flags {
		if f == "--quiet" || f == "--msgs2stderr" || f == "--no-msgs2stderr" {
			continue
		}
		if strings.HasPrefix(f, "-") && !strings.HasPrefix(f, "--") {
			f = "-" + strings.ReplaceAll(f[1:], "q", "")
			if f == "-" {
				continue
			}
		}
		out = append(out, f)
	}
	return out
}
func copiesXattrs(flags []string) bool {
	enabled := false
	for _, f := range flags {
		if f == "--no-xattrs" || f == "--no-X" {
			enabled = false
		} else if f == "--xattrs" || (strings.HasPrefix(f, "-") && !strings.HasPrefix(f, "--") && strings.Contains(f, "X")) {
			enabled = true
		}
	}
	return enabled
}
func allocationExtra(ctx context.Context, src, dst string, xattrs bool) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		_, dstErr := os.Lstat(target)
		var extra uint64
		if os.IsNotExist(dstErr) {
			if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Blocks > 0 {
				allocated := uint64(st.Blocks) * 512
				logical := uint64(0)
				if info.Mode().IsRegular() {
					logical = uint64(info.Size())
				}
				if allocated > logical {
					extra = allocated - logical
				}
			}
		} else if dstErr != nil {
			return dstErr
		}
		if xattrs {
			burden, err := changedXattrSpace(path, target)
			if err != nil {
				return err
			}
			if burden > math.MaxUint64-extra {
				return fmt.Errorf("xattr space overflow")
			}
			extra += burden
		}
		if extra > math.MaxUint64-total {
			return fmt.Errorf("allocation overflow")
		}
		total += extra
		return nil
	})
	return total, err
}
func changedXattrSpace(src, dst string) (uint64, error) {
	n, err := unix.Llistxattr(src, nil)
	if err == unix.ENOTSUP {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	names := make([]byte, n)
	n, err = unix.Llistxattr(src, names)
	if err != nil {
		return 0, err
	}
	var extra uint64
	for _, name := range strings.Split(string(names[:n]), "\x00") {
		if name == "" {
			continue
		}
		size, err := unix.Lgetxattr(src, name, nil)
		if err != nil {
			return 0, err
		}
		value := make([]byte, size)
		n, err := unix.Lgetxattr(src, name, value)
		if err != nil {
			return 0, err
		}
		value = value[:n]
		otherSize, dstErr := unix.Lgetxattr(dst, name, nil)
		same := false
		if dstErr == nil && otherSize == len(value) {
			other := make([]byte, otherSize)
			n, e := unix.Lgetxattr(dst, name, other)
			if e != nil {
				return 0, e
			}
			same = bytes.Equal(value, other[:n])
		} else if dstErr != nil && dstErr != unix.ENODATA && dstErr != unix.ENOENT && dstErr != unix.ENOTSUP {
			return 0, dstErr
		}
		if !same {
			extra += (uint64(len(name)+len(value))+4095)/4096*4096 + 4096
		}
	}
	return extra, nil
}

// All costs are collected before allocation: free-space ordering must not
// steal a folder from the disk which already contains its partial copy.
func resumeBins(ctx context.Context, disks []*domain.Disk, items []*domain.Item, reserve func(*domain.Disk) uint64, estimate transferEstimator, tick func(string)) (map[string]*domain.Bin, error) {
	bins := map[string]*domain.Bin{}
	remaining := map[string]uint64{}
	for _, d := range disks {
		if d.Free > reserve(d) {
			remaining[d.Path] = d.Free - reserve(d)
		}
	}
	for _, item := range items {
		type choice struct {
			disk            *domain.Disk
			transfer, space uint64
		}
		choices := []choice{}
		for _, d := range disks {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			tick(item.Path + " → " + d.Name)
			transfer, space, err := estimate(ctx, &domain.Command{Src: item.Location, Dst: d.Path, Entry: item.Path})
			if err != nil {
				return nil, err
			}
			choices = append(choices, choice{d, transfer, space})
		}
		sort.SliceStable(choices, func(i, j int) bool {
			if choices[i].transfer != choices[j].transfer {
				return choices[i].transfer < choices[j].transfer
			}
			if choices[i].space != choices[j].space {
				return choices[i].space < choices[j].space
			}
			return remaining[choices[i].disk.Path] > remaining[choices[j].disk.Path]
		})
		for _, ch := range choices {
			if ch.space > remaining[ch.disk.Path] {
				continue
			}
			candidate := *item
			candidate.Estimated = true
			candidate.TransferSize = ch.transfer
			candidate.RequiredSpace = ch.space
			bin := bins[ch.disk.Path]
			if bin == nil {
				bin = &domain.Bin{}
				bins[ch.disk.Path] = bin
			}
			bin.Items = append(bin.Items, &candidate)
			bin.Size += ch.transfer
			bin.RequiredSpace += ch.space
			remaining[ch.disk.Path] -= ch.space
			break
		}
	}
	return bins, nil
}

func (c *Core) otherDestinationCopies(command *domain.Command) error {
	for _, disk := range c.state.Unraid.Disks {
		root := filepath.Clean(disk.Path)
		if root == filepath.Clean(command.Src) || root == filepath.Clean(command.Dst) {
			continue
		}
		path, _, err := safeJoin(root, command.Entry)
		if err != nil {
			return err
		}
		_, err = os.Lstat(path)
		if err == nil {
			return fmt.Errorf("Copia presente anche su %s: sorgente preservata; consolidare le copie prima di MOVE", disk.Name)
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (c *Core) scatterCopyConflict(command *domain.Command) error {
	if !command.Estimated {
		return nil
	} // GATHER intentionally consolidates multiple sources.
	return c.otherDestinationCopies(command)
}
