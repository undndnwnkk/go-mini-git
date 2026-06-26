package main

import (
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/service"
	"os"
	"path/filepath"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("not enough arguments")
		return
	}

	switch args[0] {
	case "init":
		if err := os.MkdirAll(filepath.Join(".minigit", "objects"), 0755); err != nil {
			fmt.Printf("create objects dir: %v\n", err)
			return
		}

		if err := os.MkdirAll(filepath.Join(".minigit", "snapshots"), 0755); err != nil {
			fmt.Printf("create snapshots dir: %v\n", err)
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
			fmt.Printf("scan: %v\n", err)
			return
		}
	case "snapshot":
		if len(args) < 2 {
			fmt.Println("not enough arguments")
			return
		}

		data, err := service.BuildSnapshot(args[1])
		if err != nil {
			fmt.Printf("error while building snapshot: %v\n", err)
			return
		}

		err = service.SaveSnapshot(data, ".minigit/snapshots")
		if err != nil {
			fmt.Printf("error while saving snapshot: %v\n", err)
			return
		}

		fmt.Println("snapshot saved succesfully!")

	}
}
