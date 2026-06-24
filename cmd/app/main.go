package main

import (
	"fmt"
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

	}
}
