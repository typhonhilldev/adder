package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"typhonhill.dev/adder/internal/derive"
	"typhonhill.dev/adder/internal/diag"
	"typhonhill.dev/adder/internal/plan"
)

func newDeriveCommand(opts *globalOptions) *cobra.Command {
	var derived derive.Options
	var showDiff bool
	var stdout string
	cmd := &cobra.Command{
		Use:   "derive [directories]",
		Short: "Rebuild files derived from the repository's source code",
		Long: `derive runs every scan in adder.yaml and re-renders the derived outputs.

Files whose content has not changed are left untouched. With directory patterns, such as ./... or
./services/api, it runs over every adder.yaml they match. With --stdout, it prints one output file's content
instead, and writes nothing.`,
		Args: patternArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stdout != "" {
				var other string
				switch {
				case len(args) > 0:
					other = "directory patterns"
				case derived.OutDir != "":
					other = "--out"
				case derived.DryRun || showDiff:
					other = "--dry-run or --diff"
				case opts.format != formatText:
					other = "--format " + opts.format
				}
				if other != "" {
					return &usageError{fmt.Errorf("--stdout prints one file and cannot be combined with %s", other), cmd.CommandPath()}
				}
				return printOutput(cmd, opts, derived, stdout)
			}
			derived.DryRun = derived.DryRun || showDiff
			if derived.OutDir != "" && len(args) > 0 {
				return &usageError{errors.New("--out writes one configuration's outputs and cannot be combined with directory patterns"), cmd.CommandPath()}
			}
			newReport := func() report { return &deriveReport{DryRun: derived.DryRun, Changes: []changeReport{}} }
			return runConfigs(cmd, opts, args, newReport, func(run configRun, r report) error {
				rep := r.(*deriveReport)
				options := derived
				options.Root = run.root
				options.Packs = opts.packs
				p, err := derive.Run(cmd.Context(), run.cfg, options)
				if err != nil {
					return err
				}
				rep.Changes = changesReport(p.Changes, showDiff)
				if opts.json {
					return nil
				}
				out := opts.printer(cmd.OutOrStdout())
				out.prefix = run.prefix
				if err := out.scans(cmd.Context(), run.cfg, run.root); err != nil {
					return err
				}
				for _, c := range p.Changes {
					out.change(c, derived.DryRun)
				}
				if showDiff {
					out.diffs(p.Pending())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&derived.OutDir, "out", "", "write outputs under this directory instead of --root")
	cmd.Flags().BoolVarP(&derived.DryRun, "dry-run", "n", false, "show what would change without writing anything")
	cmd.Flags().BoolVar(&showDiff, "diff", false, "also show each change as a unified diff; implies --dry-run")
	cmd.Flags().StringVar(&stdout, "stdout", "", "print the content of this output file instead of writing anything")
	_ = cmd.RegisterFlagCompletionFunc("stdout", completeOutputs(opts, plan.Removed))
	return cmd
}

// printOutput prints the content derive would write to file, a path from the current directory, writing nothing.
func printOutput(cmd *cobra.Command, opts *globalOptions, options derive.Options, file string) error {
	opts.report = &deriveReport{DryRun: true, Changes: []changeReport{}}
	cfg, root, err := opts.loadFrom(filepath.Dir(file)) // the configuration nearest the file
	if err != nil {
		return err
	}
	name, err := rootRelative(root, file)
	if err != nil {
		return err
	}
	options.Root, options.Packs, options.DryRun = root, opts.packs, true
	p, err := derive.Run(cmd.Context(), cfg, options)
	if err != nil {
		return err
	}
	for _, c := range p.Changes {
		if c.Path == name && c.Status != plan.Removed {
			_, err := cmd.OutOrStdout().Write(c.New)
			return err
		}
	}
	return &diag.Diagnostic{
		Code:    diag.NoMatch,
		Pos:     diag.Pos{File: file},
		Message: "no output writes this file",
		Hint:    "adder list shows the outputs, and adder explain where a generated file comes from",
	}
}
