package core

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"unbalance/daemon/algorithm"
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
	args := append([]string{common.RsyncArgs}, flags...)
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
	return parseTransferStats(stdout.String())
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

// Estimates belong to a specific destination; never reuse them on another disk.
func resumeBin(ctx context.Context, disk *domain.Disk, items []*domain.Item, reserve uint64, estimate transferEstimator, tick func(string)) (*domain.Bin, error) {
	candidates := make([]*domain.Item, 0, len(items))
	zero := make([]*domain.Item, 0)
	originals := make(map[*domain.Item]*domain.Item, len(items))
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tick(item.Path)
		transfer, required, err := estimate(ctx, &domain.Command{Src: item.Location, Dst: disk.Path, Entry: item.Path})
		if err != nil {
			return nil, fmt.Errorf("compare %s on %s: %w", item.Path, disk.Name, err)
		}
		candidate := *item
		candidate.Estimated = true
		candidate.TransferSize = transfer
		candidate.RequiredSpace = required
		candidate.Size = required
		candidate.BlocksUsed = 0
		if required == 0 {
			zero = append(zero, &candidate)
		} else {
			candidates = append(candidates, &candidate)
		}
		originals[&candidate] = item
	}
	// Space is estimated in bytes, independent of the source filesystem blocks.
	bin := algorithm.NewKnapsack(disk, candidates, reserve, 0).BestFit()
	if bin == nil {
		if len(zero) == 0 {
			return nil, nil
		}
		bin = &domain.Bin{}
	}
	bin.Items = append(bin.Items, zero...)
	bin.RequiredSpace = bin.Size
	bin.Size = 0
	for _, item := range bin.Items {
		item.Size = originals[item].Size
		bin.Size += item.TransferSize
	}
	return bin, nil
}

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
