package main

import (
	"fmt"
	"os"

	"github.com/Izzette/kubectl-api-resource-versions/internal/cmd"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl api-resource-versions", pflag.ExitOnError)
	pflag.CommandLine = flags

	restClientGetter := genericclioptions.NewConfigFlags(true)
	ioStreams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	root := cmd.NewCmdAPIResourceVersions(restClientGetter, ioStreams)
	if err := root.Execute(); err != nil {
		if _, errWriteErr := fmt.Fprintln(ioStreams.ErrOut, err); errWriteErr != nil {
			panic(fmt.Errorf("error encountered while writing %w to %v: %w", err, ioStreams.ErrOut, errWriteErr))
		}
	}
}
