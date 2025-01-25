package main

import (
	"fysj.net/v2/conf"
	"fysj.net/v2/rpc"
	"github.com/spf13/cobra"
)

func main() {
	cobra.MousetrapHelpText = ""

	initLogger()
	initCommand()
	conf.InitConfig()
	rpc.InitRPCClients()

	setMasterCommandIfNonePresent()
	rootCmd.Execute()
}
