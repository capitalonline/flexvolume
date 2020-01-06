package utils

import "fmt"

var (
	// VERSION should be updated by hand at each release
	VERSION = "v0.1"

	// GITCOMMIT will be overwritten automatically by the build system
	GITCOMMIT = "HEAD"
)

// PluginVersion
func PluginVersion() string {
	return VERSION
}

// Usage help
func Usage() {
	fmt.Printf(
		"Use binary file as the first parameter, and format support:\n" +
			"    plugin init: \n" +
			"    plugin mount:  for nas plugin\n" +
			"    plugin umount: for nas plugin\n\n" +
			"You can refer to cds flexvolume docs: \n")
}
