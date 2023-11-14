package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "opkl-updater",
	Short: "Manages OIDC Public Key Ledger (OPKL)",
	// todo add long description: Long:  ``,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
}
