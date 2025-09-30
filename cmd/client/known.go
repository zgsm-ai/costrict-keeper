package client

import (
	"fmt"
	"os"
	"path/filepath"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/env"

	"github.com/spf13/cobra"
)

var knownCmd = &cobra.Command{
	Use:   "known",
	Short: "Output all service information to well-known.json file",
	Long:  "Collect all component, service and endpoint information and output it to specified file. If output path is not specified, default output to <user directory>/.costrict/share/.well-known.json",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		showKnowledge()
	},
}

func showKnowledge() {
	fname := filepath.Join(env.CostrictDir, "share", ".well-known.json")

	bytes, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("load '.well-known.json' failed: %v", err)
		return
	}
	fmt.Printf("%s\n", string(bytes))
}

func init() {
	root.RootCmd.AddCommand(knownCmd)
}
