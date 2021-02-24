package cmd

import (
	"fmt"
	"log"

	"github.com/DataHenHQ/till/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the Till server",
	Long:  `Starts the Till server in order to listen to and receive HTTP requests and proxy them.`,
	Run: func(cmd *cobra.Command, args []string) {
		port := viper.GetString("port")
		fmt.Println("Starting DataHen Till server on port", port)
		server.Serve(port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", "2933", "Specify the port to run")
	if err := viper.BindPFlag("port", serveCmd.Flags().Lookup("port")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}
}
