package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DataHenHQ/till/proxy"
	"github.com/DataHenHQ/till/server"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the DataHen Till server",
	Long:  `Starts the DataHen Till server in order to listen to and receive HTTP requests and proxy them.`,
	Run: func(cmd *cobra.Command, args []string) {
		port := viper.GetString("port")
		// Load or generate a new CA cert files
		caCertFile := viper.GetString("ca-cert")
		caKeyFile := viper.GetString("ca-key")
		setCaFileDefaults(&caCertFile, &caKeyFile)
		proxy.LoadOrGenCAFiles(caCertFile, caKeyFile)

		// Set UserAgent related settings
		proxy.ForceUA = viper.GetBool("force-user-agent")
		proxy.UAType = viper.GetString("ua-type")

		// start the server
		fmt.Println("Starting DataHen Till server on port", port)
		if proxy.ForceUA {
			fmt.Printf("Till is currently configured to override all User-Agent headers with random %v browsers\n", proxy.UAType)
		}
		server.Serve(port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("port", "p", "2933", "Specify the port to run")
	if err := viper.BindPFlag("port", serveCmd.Flags().Lookup("port")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("ca-cert", "", "Specify the CA certificate file (default is $HOME/.config/datahen/till/till-ca-cert.pem)")
	if err := viper.BindPFlag("ca-cert", serveCmd.Flags().Lookup("ca-cert")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("ca-key", "", "Specify the CA certificate file (default is $HOME/.config/datahen/till/till-ca-key.pem)")
	if err := viper.BindPFlag("ca-key", serveCmd.Flags().Lookup("ca-key")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().Bool("force-user-agent", true, "When set to true, will override any user-agent header with a random value based on ua-type")
	if err := viper.BindPFlag("force-user-agent", serveCmd.Flags().Lookup("force-user-agent")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("ua-type", "desktop", "Specify what kind of browser user-agent to generate. Values can either be \"desktop\" or \"mobile\"")
	if err := viper.BindPFlag("ua-type", serveCmd.Flags().Lookup("ua-type")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}
}

func setCaFileDefaults(caCertFile *string, caKeyFile *string) {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *caCertFile == "" {
		*caCertFile = filepath.Join(home, ".config", "datahen", "till", "till-ca-cert.pem")
	}

	if *caKeyFile == "" {
		*caKeyFile = filepath.Join(home, ".config", "datahen", "till", "till-ca-key.pem")
	}
}
