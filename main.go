package main

import (
    "fmt"
    "flag"
    "os"
)

type Command struct {
    Name string
    Run func(cmd *Command) error
    Flags flag.FlagSet
}

var commands = []*Command {
    cmdList,
    cmdEnable,
    cmdDisable,
    cmdRename,
}

func main() {
    flag.Parse()
    args := flag.Args()
    if len(args) < 1 {
        printUsage()
        os.Exit(1)
    }

    for _, cmd := range commands {
        if cmd.Name != args[0] {
            continue
        }

        cmd.Flags.Parse(args[1:])
        err := cmd.Run(cmd)
        if err != nil {
            fmt.Println(err)
        }
    }
}

func printUsage() {
    fmt.Fprintf(os.Stderr, "usage: linkctl [COMMAND]\n")
}
