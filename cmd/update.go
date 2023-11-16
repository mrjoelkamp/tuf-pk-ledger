package cmd

import (
	"github.com/mrjoelkamp/opkl-updater/log"
	"github.com/mrjoelkamp/opkl-updater/pkg/ledger"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the ledger from OP discovery URI",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("The provider URI argument is required")
		}
		if err := UpdateLedger(cmd, args); err != nil {
			log.Fatal(err.Error())
		}
	},
}

func UpdateLedger(cmd *cobra.Command, args []string) error {
	err := ledger.Update(args[0])
	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
