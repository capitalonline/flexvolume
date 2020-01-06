package main

import (
	"fmt"
	driver "github.com/capitalonline/flexvolume/provider/driver"
	utils "github.com/capitalonline/flexvolume/provider/utils"
	"os"
	"strings"
)

func main() {
	driver.Run()
}

// check running environment and print help
func init() {
	if len(os.Args) == 1 {
		utils.Usage()
		os.Exit(0)
	}

	argsOne := strings.ToLower(os.Args[1])
	if argsOne == "--version" || argsOne == "version" || argsOne == "-v" {
		fmt.Printf(utils.PluginVersion())
		os.Exit(0)
	}

	if argsOne == "--help" || argsOne == "help" || argsOne == "-h" {
		utils.Usage()
		os.Exit(0)
	}
}
