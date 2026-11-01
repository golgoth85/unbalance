package core

import (
	"fmt"
	"strings"
)

var blockedRsyncArgReasons = map[string]string{
	"--log-file":            "planning must not write log files",
	"--write-batch":         "batch output is outside the local transfer model",
	"--only-write-batch":    "batch output does not transfer data",
	"--read-batch":          "batch input is outside the local transfer model",
	"--remove-source-files": "unbalanced manages source deletion after validated transfers",
	"--remove-sent-files":   "unbalanced manages source deletion after validated transfers",
	"--rsync-path":          "remote rsync command execution is outside unbalanced's local transfer model",
	"--rsh":                 "remote shell execution is outside unbalanced's local transfer model",
	"-e":                    "remote shell execution is outside unbalanced's local transfer model",
}

func validateRsyncArgs(args []string) error {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}

		if arg == "--del" || arg == "--delete" || strings.HasPrefix(arg, "--delete-") {
			return fmt.Errorf("rsync option %q is not allowed because unbalanced does not perform destination mirroring/deletion", arg)
		}

		for blocked, reason := range blockedRsyncArgReasons {
			if arg == blocked || strings.HasPrefix(arg, blocked+"=") || (blocked == "-e" && strings.HasPrefix(arg, "-e")) {
				return fmt.Errorf("rsync option %q is not allowed because %s", arg, reason)
			}
		}
	}

	return nil
}

func cleanRsyncArgs(args []string) []string {
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		cleaned = append(cleaned, arg)
	}

	return cleaned
}
