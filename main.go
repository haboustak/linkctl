package main

import (
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

var IsATTY bool
var Version = "1.0.0-dev"

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

func init() {
	// HACK unix (Linux?) only
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TCGETS)
	IsATTY = err == nil
}

func main() {
	var showHelp bool
	var version bool

	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.BoolVar(&showAll, "a", false, "show all links")
	flag.BoolVar(&terseMode, "t", false, "only print link names")
	flag.BoolVar(&version, "version", false, "print version information")

	flag.Usage = func() {
		printUsage(defaultUsage)
	}
	flag.Parse()
	args := flag.Args()

	if showHelp && len(args) == 0 {
		printUsage(defaultUsage)
	} else if version {
		printVersion()
	}

	command := "list"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	cmdFound := false
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
		cmdFound = true
		break
	}

	if !cmdFound {
		fmt.Fprintf(os.Stderr, "Unknown operation \"%s\", try \"linkctl -h\"\n", command)
		os.Exit(1)
	}
}

var defaultUsage = `linkctl is a tool for managing systemd-networkd virtual interfaces

Usage:
    linkctl [-h] [-a] [-t] [-version] COMMAND [arguments]

Options:
   -a           show all links
   -h           show this help
   -t           only print link names
   -version     print version information

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

func printVersion() {
	fmt.Fprintf(os.Stdout, "linkctl %s\n", Version)
	os.Exit(0)
}
