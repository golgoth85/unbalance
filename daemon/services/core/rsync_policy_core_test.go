package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cskr/pubsub"
	"unbalance/daemon/common"

	"unbalance/daemon/domain"
)

func TestSetRsyncArgsRejectsBlockedArgAndKeepsConfig(t *testing.T) {
	for _, arg := range []string{"--remove-source-files", "--remove-sent-files", "--remove-sent-files=", "--remove-sent-files=true"} {
		t.Run(arg, func(t *testing.T) {
			c := &Core{ctx: &domain.Context{Config: domain.Config{RsyncArgs: []string{"-X"}}}}
			if _, err := c.SetRsyncArgs([]string{arg}); err == nil {
				t.Fatal("source deletion option accepted")
			}
			if len(c.ctx.Config.RsyncArgs) != 1 || c.ctx.Config.RsyncArgs[0] != "-X" {
				t.Fatalf("config changed after rejected arg: %v", c.ctx.Config.RsyncArgs)
			}
		})
	}
}

func TestStoredSourceRemovalAliasCannotStartAnyOperation(t *testing.T) {
	for _, arg := range []string{"--remove-sent-files", "--remove-sent-files=", "--remove-sent-files=true"} {
		for _, mode := range []string{"Copy", "Move", "Gather", "Validate", "Replay"} {
			t.Run(arg+"/"+mode, func(t *testing.T) {
				src, dst := t.TempDir(), t.TempDir()
				writeResumeFile(t, src, "share/file", "source must survive")
				hub := pubsub.New(4)
				t.Cleanup(hub.Shutdown)
				events := hub.Sub("socket:broadcast")
				operation := &domain.Operation{
					OpKind:    common.OpScatterCopy,
					RsyncArgs: []string{common.RsyncArgs, arg},
					Commands:  []*domain.Command{{Src: src, Dst: dst, Entry: "share", Status: common.CmdPending}},
				}
				c := &Core{ctx: &domain.Context{Hub: hub}, state: &domain.State{}}
				switch mode {
				case "Move":
					operation.OpKind = common.OpScatterMove
				case "Gather":
					operation.OpKind = common.OpGatherMove
				case "Validate":
					operation = c.createScatterValidateOperation(*operation)
				case "Replay":
					operation = c.createReplayOperation(*operation)
				}
				c.state.Operation = operation
				c.setStatus(operation.OpKind)
				c.armOperation()
				comparison := c.currentComparisonContext()
				c.runOperation(mode)
				if c.state.Operation != nil || c.state.Status != common.OpNeutral || !operation.Started.IsZero() {
					t.Fatal("blocked operation started or remained active")
				}
				if comparison.Err() == nil {
					t.Fatal("blocked operation kept comparison context alive")
				}
				select {
				case event := <-events:
					packet, ok := event.(*domain.Packet)
					if !ok || packet.Topic != common.EventOperationError || !strings.Contains(fmt.Sprint(packet.Payload), "--remove-sent-files") {
						t.Fatal("expected policy error before transfer events", event)
					}
				case <-time.After(time.Second):
					t.Fatal("policy error not published")
				}
				data, err := os.ReadFile(filepath.Join(src, "share/file"))
				if err != nil || string(data) != "source must survive" {
					t.Fatal("source changed", err)
				}
				if _, err := os.Lstat(filepath.Join(dst, "share")); !os.IsNotExist(err) {
					t.Fatal("destination changed", err)
				}
			})
		}
	}
}

func TestRemovalAliasIsRejectedBeforeComparisonStartsRsync(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeResumeFile(t, src, "share/file", "source must survive")
	for _, arg := range []string{"--remove-sent-files", "--remove-sent-files=", "--remove-sent-files=true"} {
		_, _, err := estimateTransfer(context.Background(), "/must-not-start-rsync", &domain.Command{Src: src, Dst: dst, Entry: "share"}, []string{arg})
		if err == nil || !strings.Contains(err.Error(), "source deletion after validated transfers") {
			t.Fatal("expected policy rejection before process startup", arg, err)
		}
	}
}
