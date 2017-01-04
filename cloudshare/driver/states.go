package driver

import (
	cs "github.com/cloudshare/go-sdk/cloudshare"
	"github.com/docker/machine/libmachine/state"
)

func ToDockerMachineState(code cs.EnvironmentStatusCode) state.State {
	switch code {
	case cs.StatusReady:
		return state.Running
	case cs.StatusPreparing:
		return state.Starting
	case cs.StatusDeleted:
		return state.None
	case cs.StatusAllocationScheduledNoRun:
		return state.Starting
	case cs.StatusSuspended:
		return state.Paused
	case cs.StatusArchived:
		return state.None
	case cs.StatusPublishing:
		return state.Running
	case cs.StatusCreationFailed:
		return state.Error
	case cs.StatusInGrace:
		return state.None
	case cs.StatusStopping:
		return state.Stopping
	default:
		return state.None
	}
}
