package main

import (
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/service"
	"os"
)

func main() {
	args := os.Args[1:]

	switch args[0] {
	case "init":
		err := os.MkdirAll(".minigit/objects", 0755)
		if err != nil {
			if os.IsExist(err) {
				fmt.Println(".minigit already initialized")
			} else {
				fmt.Printf("unknown error: %v", err)
				return
			}
		}

		err = os.Mkdir(".minigit/snapshots", 0755)
		if err != nil && !os.IsExist(err) {
			fmt.Printf("unknown error: %v", err)
			return
		}

		fmt.Println(".minigit folder created")
	case "scan":
		if len(args) < 2 {
			fmt.Println("not enough arguments")
			return
		}

		err := service.Scan(args[1])
		if err != nil {
			fmt.Errorf("scan: %w", err)
		}

	}
}
