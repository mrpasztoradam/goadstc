package main

import (
	"fmt"

	"github.com/mrpasztoradam/goadstc"
)

func main() {
	// Simple version
	fmt.Println("Version:", goadstc.Version())

	// Detailed build info
	info := goadstc.GetBuildInfo()
	fmt.Println("\nBuild Information:")
	fmt.Println(info.String())

	// Individual fields
	fmt.Println("\nDetailed:")
	fmt.Printf("  Version:    %s\n", info.Version)
	fmt.Printf("  Go Version: %s\n", info.GoVersion)
	if info.GitCommit != "" {
		fmt.Printf("  Git Commit: %s\n", info.GitCommit)
		if info.Dirty {
			fmt.Println("  Status:     dirty (uncommitted changes)")
		}
	}
	if info.GitTag != "" {
		fmt.Printf("  Git Tag:    %s\n", info.GitTag)
	}
	if info.BuildTime != "" {
		fmt.Printf("  Build Time: %s\n", info.BuildTime)
	}
}
