package cmd

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DataHenHQ/license"
	"github.com/DataHenHQ/till/internal/tillclient"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var tillHomeDir string
var BaseURL = "https://till.datahen.com/api/v1"
var PubKey = "ca60c6f94f2ff9f030e4636e66e018fe4f930a16e8915920f390b9bcff9adf9f"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "till",
	Short: "DataHen Till empowers your existing web scraper to be unblockable, scalable and maintainable without code changes",
	Long: `DataHen Till is a standalone tool that runs alongside your web scraper, 
and instantly makes your existing web scraper unblockable, scalable, and maintainable, 
without requiring any code changes on your scraper code.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// set the base url
	tillclient.BaseURL = BaseURL

	// set the license's public key
	decpubkey, err := hex.DecodeString(string(PubKey))
	if err != nil {
		log.Fatalln("could not decode public key:", PubKey)
	}
	license.PublicKey = decpubkey

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	initTillVersion()
	initTillHomeDir()
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %v)", filepath.Join(tillHomeDir, "config.yaml")))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initTillVersion() {

}

func initTillHomeDir() {
	userhome, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Search config in home directory with name ".till" (without extension).
	tillHomeDir = filepath.Join(userhome, ".config", "datahen", "till")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		viper.AddConfigPath(tillHomeDir)
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
