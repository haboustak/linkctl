package main

import (
	"flag"
	"fmt"
	"os"
)

type Command struct {
	Name  string
	Run   func(cmd *Command) error
	Flags flag.FlagSet
    Usage string
}

var commands = []*Command{
	cmdList,
	cmdEnable,
	cmdDisable,
	cmdRename,
}

func main() {
    var showHelp bool

    flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()
	args := flag.Args()

	if showHelp && len(args) == 0 {
        printUsage(defaultUsage)
	}

    command := "list"
    if len(args) > 0 {
        command = args[0]
        args = args[1:]
    }

	for _, cmd := range commands {
		if cmd.Name != command {
			continue
		}

        if showHelp {
            printUsage(cmd.Usage)
        }

        cmd.Flags.Usage = func() {
            printUsage(cmd.Usage)
        }
		cmd.Flags.Parse(args)
		err := cmd.Run(cmd)
		if err != nil {
			fmt.Println(err)
		}
	}
}

var defaultUsage = `linkctl is a tool for managing systemd-networkd virtual interfaces

Usage:
    linkctl [-h] COMMAND [arguments]

Commands:
    list        list netdev links
    enable      enable a netdev link
    disable     disable a netdev link
    rename      rename a netdev link
`
func printUsage(usage string) {
    fmt.Fprintf(os.Stderr, usage)
    os.Exit(1)
}
