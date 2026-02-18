package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/steffakasid/kubectl-co/internal"
)

func isCompletionInvocation() bool {
	if len(os.Args) > 1 && os.Args[1] == "completion" {
		return false
	}
	return os.Getenv("COMP_LINE") != "" && os.Getenv("COMP_POINT") != ""
}

func handleCompletion() {
	line := os.Getenv("COMP_LINE")
	point := len(line)
	if pointStr := os.Getenv("COMP_POINT"); pointStr != "" {
		parsed, err := strconv.Atoi(pointStr)
		if err == nil && parsed >= 0 && parsed <= len(line) {
			point = parsed
		}
	}
	line = line[:point]
	words := strings.Fields(line)

	cur := ""
	if len(line) > 0 {
		last := line[len(line)-1]
		if last != ' ' && last != '\t' && len(words) > 0 {
			cur = words[len(words)-1]
		}
	}

	flags := []string{"--add", "--delete", "--previous", "--current", "--debug", "--help", "--version"}
	if strings.HasPrefix(cur, "-") {
		for _, flag := range flags {
			if strings.HasPrefix(flag, cur) {
				fmt.Println(flag)
			}
		}
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	completionCO, err := internal.NewCO(home)
	if err != nil {
		return
	}

	err = completionCO.ListConfigs()
	if err != nil {
		return
	}

	printConfigs(completionCO)
}

func handleCompletionCommand(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: kubectl-co completion bash|zsh")
		return
	}

	switch args[0] {
	case "bash":
		fmt.Println("complete -C kubectl-co kubectl-co")
		fmt.Println("complete -C kubectl-co kubectl")
	case "zsh":
		fmt.Println("autoload -U +X bashcompinit && bashcompinit")
		fmt.Println("complete -C kubectl-co kubectl-co")
		fmt.Println("complete -C kubectl-co kubectl")
	default:
		fmt.Fprintln(os.Stderr, "Unsupported shell. Use 'bash' or 'zsh'.")
	}
}
