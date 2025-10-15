package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	watchPath := flag.String("watch", "", "path to watch")
	backupPath := flag.String("backup", "", "path to backup")
	refreshInterval := flag.Int("refresh", 60, "scan interval in seconds")
	restorePath := flag.String("restore", "", "path to restored files")

	flag.Parse()

	if *watchPath != "" {
		if *backupPath == "" {
			log.Println("Error: --backup required for watch mode")
			fmt.Println("\nUsage:")
			fmt.Println("  ./app --watch <path> --backup <path> --refresh <seconds>")
			os.Exit(1)
		}
		if err := watch(*watchPath, *backupPath, *refreshInterval); err != nil {
			log.Fatal(err)
		}
	} else if *restorePath != "" {
		if *backupPath == "" {
			log.Println("Error: --backup required for restore mode")
			fmt.Println("\nUsage:")
			fmt.Println("  ./app --restore <path> --backup <path>")
			os.Exit(1)
		}
		if err := restore(*backupPath, *restorePath); err != nil {
			log.Fatal(err)
		}
	} else {
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "watch" && *watchPath == "" {
				log.Println("Error: --watch requires a path")
				fmt.Println("\nUsage:")
				fmt.Println("  ./app --watch <path> --backup <path> --refresh <seconds>")
				os.Exit(1)
			}
			if f.Name == "restore" && *restorePath == "" {
				log.Println("Error: --restore requires a path")
				fmt.Println("\nUsage:")
				fmt.Println("  ./app --restore <path> --backup <path>")
				os.Exit(1)
			}
		})
	}
}
