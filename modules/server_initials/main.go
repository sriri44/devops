package main

import (
	"fmt"
	"os/exec"
)

func main() {
	commands := [][]string{
		{"sudo", "apt", "install", "git", "-y"},
		{"sudo", "apt", "install", "golang-go", "-y"},
		{"sudo", "apt", "install", "nginx", "-y"},
	}
	var gitCloneURL string
	fmt.Print("Enter the git clone URL: ")
	fmt.Scanln(&gitCloneURL)
	commands = append(commands, []string{"git", "clone", gitCloneURL})

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		fmt.Printf("Running: %v\n", cmdArgs)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error running %v: %v\n", cmdArgs, err)
			return
		}
	}

	fmt.Println("All commands executed successfully.")
}
