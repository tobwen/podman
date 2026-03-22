package volumes

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/containers/podman/v6/cmd/podman/common"
	"github.com/containers/podman/v6/cmd/podman/parse"
	"github.com/containers/podman/v6/cmd/podman/registry"
	"github.com/containers/podman/v6/cmd/podman/utils"
	"github.com/containers/podman/v6/cmd/podman/validate"
	"github.com/containers/podman/v6/pkg/domain/entities"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/completion"
)

var (
	volumePruneDescription = `By default, only anonymous (unnamed) unused volumes are removed.
  Use --all to remove all unused volumes (anonymous and named).
  The command prompts for confirmation which can be overridden with the --force flag.`
	pruneCommand = &cobra.Command{
		Use:               "prune [options]",
		Args:              validate.NoArgs,
		Short:             "Remove unused volumes",
		Long:              volumePruneDescription,
		RunE:              prune,
		ValidArgsFunction: completion.AutocompleteNone,
	}
	filter = []string{}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: pruneCommand,
		Parent:  volumeCmd,
	})
	flags := pruneCommand.Flags()

	filterFlagName := "filter"
	flags.StringArrayVar(&filter, filterFlagName, []string{}, "Provide filter values (e.g. 'label=<key>=<value>')")
	_ = pruneCommand.RegisterFlagCompletionFunc(filterFlagName, common.AutocompleteVolumePruneFilters)
	flags.BoolP("force", "f", false, "Do not prompt for confirmation")
	flags.BoolP("all", "a", false, "Remove all unused volumes, both anonymous and named")
	flags.Bool("include-pinned", false, "Include pinned volumes in prune operation")
}

func prune(cmd *cobra.Command, _ []string) error {
	var (
		pruneOptions  = entities.VolumePruneOptions{}
		listOptions   = entities.VolumeListOptions{}
		unusedOptions = entities.VolumeListOptions{}
	)
	ctx := registry.Context()
	// Prompt for confirmation if --force is not set
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	pruneOptions.Filters, err = parse.FilterArgumentsIntoFilters(filter)
	if err != nil {
		return err
	}

	// --all adds filter all=true (Docker-compatible; behavior is filter-only)
	allFlag, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	effectiveAll := allFlag || isTruthyFilter(pruneOptions.Filters, "all")
	if effectiveAll {
		pruneOptions.Filters.Set("all", "true")
	}

	includePinned, err := cmd.Flags().GetBool("include-pinned")
	if err != nil {
		return err
	}
	pruneOptions.IncludePinned = includePinned
	if !force {
		reader := bufio.NewReader(os.Stdin)
		if effectiveAll {
			fmt.Println("WARNING! This will remove all volumes not used by at least one container. The following volumes will be removed:")
		} else {
			fmt.Println("WARNING! This will remove anonymous local volumes not used by at least one container. The following volumes will be removed:")
		}
		listOptions.Filter, err = parse.FilterArgumentsIntoFilters(filter)
		if err != nil {
			return err
		}

		// Exclude pinned volumes from the confirmation preview unless
		// --include-pinned was requested. The actual prune call also
		// receives IncludePinned as the authoritative behavior flag.
		if !includePinned {
			if listOptions.Filter == nil {
				listOptions.Filter = make(map[string][]string, 1)
			}
			if _, hasPinnedFilter := listOptions.Filter["pinned"]; !hasPinnedFilter {
				listOptions.Filter["pinned"] = []string{"false"}
			}
		}

		filteredVolumes, err := registry.ContainerEngine().VolumeList(ctx, listOptions)
		if err != nil {
			return err
		}
		var finalVolumes []*entities.VolumeListReport
		if effectiveAll {
			unusedOptions.Filter = map[string][]string{"dangling": {"true"}}
			unusedVolumes, err := registry.ContainerEngine().VolumeList(ctx, unusedOptions)
			if err != nil {
				return err
			}
			finalVolumes = getIntersection(unusedVolumes, filteredVolumes)
		} else {
			danglingOptions := entities.VolumeListOptions{Filter: map[string][]string{"dangling": {"true"}}}
			anonymousOptions := entities.VolumeListOptions{Filter: map[string][]string{"anonymous": {"true"}}}
			danglingVolumes, err := registry.ContainerEngine().VolumeList(ctx, danglingOptions)
			if err != nil {
				return err
			}
			anonymousVolumes, err := registry.ContainerEngine().VolumeList(ctx, anonymousOptions)
			if err != nil {
				return err
			}
			finalVolumes = getIntersection(getIntersection(danglingVolumes, anonymousVolumes), filteredVolumes)
		}
		if len(finalVolumes) < 1 {
			if effectiveAll {
				fmt.Println("No dangling volumes found")
			} else {
				fmt.Println("No dangling anonymous volumes found")
			}
			return nil
		}
		for _, fv := range finalVolumes {
			fmt.Println(fv.Name)
		}
		fmt.Print("Are you sure you want to continue? [y/N] ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.ToLower(answer)[0] != 'y' {
			return nil
		}
	}
	responses, err := registry.ContainerEngine().VolumePrune(ctx, pruneOptions)
	if err != nil {
		return err
	}
	return utils.PrintVolumePruneResults(responses, false)
}

func isTruthyFilter(filters url.Values, key string) bool {
	if filters == nil {
		return false
	}
	for _, value := range filters[key] {
		switch strings.ToLower(value) {
		case "true", "1", "y", "yes":
			return true
		}
	}
	return false
}

func getIntersection(a, b []*entities.VolumeListReport) []*entities.VolumeListReport {
	var intersection []*entities.VolumeListReport
	hash := make(map[string]bool, len(a))
	for _, aa := range a {
		hash[aa.Name] = true
	}
	for _, bb := range b {
		if hash[bb.Name] {
			intersection = append(intersection, bb)
		}
	}
	return intersection
}
