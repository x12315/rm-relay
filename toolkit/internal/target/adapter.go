// Package target defines the handoff from a verified Build Output to a development target.
package target

import (
	"context"

	"github.com/x12315/rm-relay/toolkit/internal/buildoutput"
	"github.com/x12315/rm-relay/toolkit/internal/profile"
)

// FlashRequest identifies a verified artifact and one Profile target capability.
type FlashRequest struct {
	BuildOutput buildoutput.Verified
	Profile     profile.Loaded
	TargetName  string
	DryRun      bool
}

// FlashResult records the native command and whether it was executed.
type FlashResult struct {
	Command  []string
	Executed bool
}

// Adapter sends a verified Build Output to one kind of target.
type Adapter interface {
	Flash(context.Context, FlashRequest) (FlashResult, error)
}
