package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DataHenHQ/till/proxy"
	"github.com/DataHenHQ/till/server"
	"github.com/DataHenHQ/tillup/cache"
	"github.com/DataHenHQ/tillup/interceptors"
	"github.com/DataHenHQ/tillup/logger"
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
		apiport := viper.GetString("apiport")
		proxy.ReleaseVersion = ReleaseVersion
		// Load or generate a new CA cert files
		caCertFile := viper.GetString("ca-cert")
		caKeyFile := viper.GetString("ca-key")
		setCaFileDefaults(&caCertFile, &caKeyFile)
		proxy.LoadOrGenCAFiles(caCertFile, caKeyFile)

		// Set UserAgent related settings
		proxy.ForceUA = viper.GetBool("force-user-agent")
		proxy.UAType = viper.GetString("ua-type")
		if proxy.ForceUA {
			fmt.Printf("Till is currently configured to override all User-Agent headers with random %v browsers\n", proxy.UAType)
		}

		// set the Token
		token := viper.GetString("token")
		if token == "" {
			fmt.Println("You need to specify the Till auth token. To get your token, sign up for free at https://till.datahen.com")
			fmt.Println("aborting server")
			return
		}
		server.Token = token
		proxy.Token = token

		// set the instance name
		instance := viper.GetString("instance")
		if instance == "" {
			fmt.Println("You need to specify the name of this Till instance.")
			fmt.Println("aborting server")
			return
		}
		server.InstanceName = instance
		proxy.InstanceName = instance

		// set the proxy-file
		proxyFile := viper.GetString("proxy-file")
		if proxyFile != "" {
			count, urls, err := proxy.LoadProxyFile(proxyFile)
			if err != nil {
				fmt.Println("Problem loading the proxy-file:", err)
				fmt.Println("aborting server")
				return
			}
			if count == 0 {
				fmt.Println("The supplied proxy-file does not contain any proxies. Please supply a correct proxy-file")
				fmt.Println("aborting server")
				return
			}

			// set the proxy urls and counts
			server.ProxyURLs = urls
			proxy.ProxyURLs = urls
			server.ProxyCount = count
			proxy.ProxyCount = count

			fmt.Printf("Using proxy-file to randomize through %d proxies: %v\n", count, proxyFile)
		} else {
			fmt.Println("Warning! No proxy-file supplied. You will be exposing your own IP address if you don't use Till with a proxy", proxyFile)
		}

		// sets the DB path
		dbpath := viper.GetString("dbpath")
		if dbpath == "" {
			dbpath = filepath.Join(tillHomeDir, fmt.Sprintf("%v.db", instance))
		}
		server.DBPath = dbpath

		// sets the interceptors
		var rs []interceptors.Interceptor
		viper.UnmarshalKey("interceptors", &rs)
		if rs != nil {
			// validates the interceptors
			if ok, errs := interceptors.ValidateAll(rs); !ok || len(errs) > 0 {
				log.Fatal("Your config file has invalid interceptors:", errs)
			}

			server.Interceptors = rs
		}

		// Sets cache related configurations
		var cacheconf cache.Config
		viper.UnmarshalKey("cache", &cacheconf)
		if _, err := cacheconf.Validate(); err != nil {
			log.Fatal("Your config file has invalid cache settings:", err)
		}
		cacheconf.SetDefaults()
		server.Cache = cacheconf
		proxy.Cache = cacheconf

		// Set Har settings
		proxy.HAR = viper.GetBool("har")
		proxy.HAROutput = viper.GetString("har-output")

		// Sets logger related configurations
		var loggerconf logger.Config
		viper.UnmarshalKey("logger", &loggerconf)
		if _, err := loggerconf.Validate(); err != nil {
			log.Fatal("Your config file has invalid logger settings:", err)
		}
		loggerconf.SetDefaults()
		proxy.LoggerConfig = loggerconf
		server.LoggerConfig = loggerconf

		// start the server
		server.Serve(port, apiport)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("token", "t", "", "Specify the Till auth token. To get your token, sign up for free at https://www.datahen.com/till")
	if err := viper.BindPFlag("token", serveCmd.Flags().Lookup("token")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().StringP("instance", "i", "default", "Specify the name of the Till instance.")
	if err := viper.BindPFlag("instance", serveCmd.Flags().Lookup("instance")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().StringP("port", "p", "2933", "Specify the port to run")
	if err := viper.BindPFlag("port", serveCmd.Flags().Lookup("port")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("apiport", "2980", "Specify the port to run the API server")
	if err := viper.BindPFlag("apiport", serveCmd.Flags().Lookup("apiport")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("ca-cert", "", fmt.Sprintf("Specify the CA certificate file (default is %v)", filepath.Join(tillHomeDir, "till-ca-cert.pem")))
	if err := viper.BindPFlag("ca-cert", serveCmd.Flags().Lookup("ca-cert")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("ca-key", "", fmt.Sprintf("Specify the CA certificate file (default is %v)", filepath.Join(tillHomeDir, "till-ca-key.pem")))
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

	serveCmd.Flags().String("proxy-file", "", "Specify the path to a txt file that contains a list of proxies")
	if err := viper.BindPFlag("proxy-file", serveCmd.Flags().Lookup("proxy-file")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("dbpath", "", "Specify the path to the DB that Till uses")
	if err := viper.BindPFlag("dbpath", serveCmd.Flags().Lookup("dbpath")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().Bool("har", false, "When set to true, will log requests in HAR format")
	if err := viper.BindPFlag("har", serveCmd.Flags().Lookup("har")); err != nil {
		log.Fatal("Unable to bind flag:", err)
	}

	serveCmd.Flags().String("har-output", "", "Specify the path to the HAR output log that Till will save to")
	if err := viper.BindPFlag("har-output", serveCmd.Flags().Lookup("har-output")); err != nil {
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
