package datahubdeployment

import (
	"fmt"
	"github.com/mimiro-io/datahub-config-deployment/internal/app"
	"github.com/mimiro-io/datahub-config-deployment/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "mim-deploy",
	Short: "MIMIRO Data Hub configuration deployment CLI",
	Long:  `MIMIRO Data Hub configuration deployment CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		silent, _ := cmd.Flags().GetBool("silent")
		if silent {
			pterm.DisableOutput()
		}

		app, err := app.NewApp(cmd, args)
		utils.HandleError(err)

		err = app.Run()
		utils.HandleError(err)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().StringP("datahub", "d", "", "Datahub server URL")
	RootCmd.Flags().String("token", "", "Signin Bearer token to use against the DataHub")
	RootCmd.Flags().StringP("path", "p", "", "Root path of the config location")
	RootCmd.Flags().StringP("env", "e", "", "Variable file to use for substitution")
	RootCmd.Flags().StringP("log-format", "l", "", "Log format to use when executing mim commands")
	RootCmd.Flags().Bool("dry-run", true, "If set to true, only test the changes without applying them")
	RootCmd.Flags().Bool("create-manifest", true, "Should create a manifest if it is missing")
	RootCmd.Flags().Bool("abort-missing-secret", true, "Should abort if secret is missing")
	RootCmd.Flags().Bool("token-stdin", false, "If true, expects a Bearer token on StdIn")
	RootCmd.Flags().Bool("silent", false, "Enable to silence output")
	RootCmd.Flags().Bool("display-manifest", false, "Enable to output the Manifest")
	RootCmd.Flags().Bool("json", false, "Enable to make Manifest output json compatible")
}
