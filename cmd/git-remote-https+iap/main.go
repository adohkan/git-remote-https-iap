package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	BinaryName = "git-remote-https+iap"
)

var (
	rootCmd = &cobra.Command{
		Use:   fmt.Sprintf("%s remote url", BinaryName),
		Short: "git-remote-helper that handles authentication for GCP Identity Aware Proxy",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			remote, url := args[0], args[1]
			log.Debug().Msgf("%s %s %s", BinaryName, remote, url)
		},
	}
)

func init() {
	// set default log level
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
