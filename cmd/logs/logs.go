package logs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"costrict-keeper/cmd/root"
	"costrict-keeper/services"
)

var (
	logFile     string
	serviceName string
	directory   string
	logService  *services.LogService
)

func init() {
	root.RootCmd.AddCommand(Cmd)
	Cmd.Flags().SortFlags = false
	Cmd.Flags().StringVarP(&logFile, "file", "f", "", "Log file path")
	Cmd.Flags().StringVarP(&serviceName, "service", "s", "", "Service name")
	Cmd.Flags().StringVarP(&directory, "directory", "d", "", "Log directory path")
}

var Cmd = &cobra.Command{
	Use:   "logs",
	Short: "Report logs to the cloud",
	Run: func(cmd *cobra.Command, args []string) {
		if logFile == "" && directory == "" {
			fmt.Println("Please use -f parameter to specify log file or use -d parameter to specify log directory")
			return
		}
		logService = services.NewLogService(viper.GetViper())

		if directory != "" {
			// Upload all log files in the directory
			dest, err := logService.UploadLogDirectory(directory, serviceName)
			if err != nil {
				fmt.Printf("Failed to upload log directory: %v\n", err)
				return
			}
			fmt.Printf("Upload successful: %s -> %s\n", directory, dest)
		} else {
			// Upload single log file
			dest, err := logService.UploadLog(logFile, serviceName)
			if err != nil {
				fmt.Printf("Failed to upload log: %v\n", err)
				return
			}
			fmt.Printf("Upload successful: %s -> %s\n", logFile, dest)
		}
	},
}
