//go:build !remote && (linux || freebsd)

package abi

import (
	"net/url"
	"testing"

	"github.com/containers/podman/v6/pkg/domain/entities"
)

func TestVolumePruneOptionsFromSystemPruneOptionsIncludesPinned(t *testing.T) {
	filters := map[string][]string{
		"label": {"keep=true"},
	}
	systemPruneOptions := entities.SystemPruneOptions{
		Filters: filters,
	}
	systemPruneOptions.VolumePruneOptions.IncludePinned = true

	volumePruneOptions := entities.VolumePruneOptions{
		Filters:       (url.Values)(systemPruneOptions.Filters),
		IncludePinned: systemPruneOptions.VolumePruneOptions.IncludePinned,
	}

	if !volumePruneOptions.IncludePinned {
		t.Fatalf("expected IncludePinned to be propagated")
	}

	if got := volumePruneOptions.Filters.Get("label"); got != "keep=true" {
		t.Fatalf("expected label filter to be propagated, got %q", got)
	}
}
