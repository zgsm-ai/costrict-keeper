/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package tunnel

import (
	"costrict-keeper/cmd/root"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel operations (list, start/stop etc.)",
	Long:  `Tunnel operations (list, start/stop etc.)`,
}

const tunnelExample = `  # start tunnel
  costrict tunnel start codebase-indexer`

func init() {
	root.RootCmd.AddCommand(tunnelCmd)

	tunnelCmd.Example = tunnelExample
}
