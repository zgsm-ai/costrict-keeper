package client

import (
	"context"
	"fmt"

	"costrict-keeper/cmd/root"
	"costrict-keeper/internal/rpc"

	"github.com/spf13/cobra"
)

var knownCmd = &cobra.Command{
	Use:   "known",
	Short: "Output all service information to well-known.json file",
	Long:  "Collect all component, service and endpoint information and output it to specified file. If output path is not specified, default output to <user directory>/.costrict/share/.well-known.json",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		exportKnowledgeViaRPC(context.Background())
	},
}

/**
* Export knowledge via RPC connection to costrict server
* @param {context.Context} ctx - Context for request cancellation and timeout
* @param {string} customOutputPath - Custom output file path, if empty uses default path
* @returns {error} Returns error if export fails, nil on success
* @description
* - Creates new RPC client instance
* - Calls POST /costrict/api/v1/known endpoint to export knowledge
* - Handles connection errors and API response errors
* - Outputs the exported knowledge file path
* - Used for remote knowledge export via costrict server
* @throws
* - Connection establishment errors
* - API request errors
* - Response parsing errors
* @example
* err := exportKnowledgeViaRPC(context.Background())
* if err != nil {
*     log.Fatal(err)
* }
 */
func exportKnowledgeViaRPC(ctx context.Context) error {
	rpcClient := rpc.NewHTTPClient(nil)

	// 调用 costrict 的 RESTful API POST 方法
	resp, err := rpcClient.Post("/costrict/api/v1/known", nil)
	if err != nil {
		return fmt.Errorf("Failed to call costrict API: %v", err)
	}

	// 检查HTTP状态码是否在200-299范围内
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		if resp.Body != nil {
			if message, msgExists := resp.Body["message"]; msgExists {
				if messageStr, ok := message.(string); ok {
					fmt.Printf("Successfully exported knowledge: %s\n", messageStr)

					// 如果响应中包含文件路径，输出它
					if path, pathExists := resp.Body["path"]; pathExists {
						if pathStr, ok := path.(string); ok {
							fmt.Printf("Knowledge exported to: %s\n", pathStr)
						}
					}
					return nil
				}
			}
		}
		// 即使没有message字段，只要状态码在200-299范围内，也认为成功
		fmt.Printf("Successfully exported knowledge, status code: %d\n", resp.StatusCode)
		return nil
	}

	// 如果响应中包含错误信息
	if resp.Body != nil {
		if errorMsg, exists := resp.Body["error"]; exists {
			if errorStr, ok := errorMsg.(string); ok {
				return fmt.Errorf("Costrict API returned error: %s", errorStr)
			}
		}
	}

	return fmt.Errorf("Unexpected response from costrict API, status code: %d", resp.StatusCode)
}

func init() {
	root.RootCmd.AddCommand(knownCmd)
}
