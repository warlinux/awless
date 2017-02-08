package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wallix/awless/cloud/aws"
	"github.com/wallix/awless/console"
	"github.com/wallix/awless/database"
	"github.com/wallix/awless/graph"
	"github.com/wallix/awless/sync"
	"github.com/wallix/awless/sync/repo"
)

var (
	showProperties bool
)

func init() {
	RootCmd.AddCommand(historyCmd)

	historyCmd.Flags().BoolVarP(&showProperties, "properties", "p", false, "Full diff with resources properties")
}

var historyCmd = &cobra.Command{
	Use:                "history",
	Short:              "Show your infrastucture history",
	PersistentPreRun:   applyHooks(initAwlessEnvHook, initCloudServicesHook, initSyncerHook, checkStatsHook),
	PersistentPostRunE: saveHistoryHook,

	RunE: func(cmd *cobra.Command, args []string) error {
		if !repo.IsGitInstalled() {
			fmt.Printf("No history available. You need to install git")
			os.Exit(0)
		}

		region := database.MustGetDefaultRegion()

		root := graph.InitResource(region, graph.Region)

		var diffs []*sync.Diff

		all, err := sync.DefaultSyncer.List()
		exitOn(err)

		for i := 1; i < len(all); i++ {
			from, err := sync.DefaultSyncer.LoadRev(all[i-1].Id)
			exitOn(err)

			to, err := sync.DefaultSyncer.LoadRev(all[i].Id)
			exitOn(err)

			d, err := sync.BuildDiff(from, to, root)
			exitOn(err)

			diffs = append(diffs, d)
		}

		for _, diff := range diffs {
			displayRevisionDiff(diff, aws.AccessService.Name(), root, verboseFlag)
			displayRevisionDiff(diff, aws.InfraService.Name(), root, verboseFlag)
		}

		return nil
	},
}

func displayRevisionDiff(diff *sync.Diff, cloudService string, root *graph.Resource, verbose bool) {
	fromRevision := "repository creation"
	if diff.From.Id != "" {
		fromRevision = diff.From.Id[:7] + " on " + diff.From.Date.Format("Monday January 2, 15:04")
	}

	var graphdiff *graph.Diff
	if cloudService == aws.InfraService.Name() {
		graphdiff = diff.InfraDiff
	}
	if cloudService == aws.AccessService.Name() {
		graphdiff = diff.AccessDiff
	}

	if showProperties {
		if graphdiff.HasDiff() {
			fmt.Println("▶", cloudService, "properties, from", fromRevision,
				"to", diff.To.Id[:7], "on", diff.To.Date.Format("Monday January 2, 15:04"))
			displayer := console.BuildOptions(
				console.WithFormat("table"),
				console.WithRootNode(root),
			).SetSource(graphdiff).Build()
			exitOn(displayer.Print(os.Stdout))
			fmt.Println()
		} else if verbose {
			fmt.Println("▶", cloudService, "properties, from", fromRevision,
				"to", diff.To.Id[:7], "on", diff.To.Date.Format("Monday January 2, 15:04"))
			fmt.Println("No changes.")
		}
	} else {
		if graphdiff.HasDiff() {
			fmt.Println("▶", cloudService, "resources, from", fromRevision,
				"to", diff.To.Id[:7], "on", diff.To.Date.Format("Monday January 2, 15:04"))
			displayer := console.BuildOptions(
				console.WithFormat("tree"),
				console.WithRootNode(root),
			).SetSource(graphdiff).Build()
			exitOn(displayer.Print(os.Stdout))
			fmt.Println()
		} else if verbose {
			fmt.Println("▶", cloudService, "resources, from", fromRevision,
				"to", diff.To.Id[:7], "on", diff.To.Date.Format("Monday January 2, 15:04"))
			fmt.Println("No resource changes.")
		}
	}
}
