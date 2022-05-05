package cmd

import (
	"github.com/spf13/cobra"
)

var (
	outputJson bool
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [options] <key>",
		Short: "get value by key",
		Run:   getCmdFunc,
	}

	cmd.Flags().BoolVar(&outputJson, "json", false, "Output as json string")

	return cmd
}

func getCmdFunc(cmd *cobra.Command, args []string) {

}
