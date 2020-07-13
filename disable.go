package main

import (
    "errors"
    "fmt"

    "github.com/haboustak/linkctl/internal/networkd"
)

var cmdDisable = &Command {
    Name:   "disable",
    Run:    disable,
}

func init() {
}

func disable(self *Command) error {
    args := self.Flags.Args()

    if len(args) != 1 {
        fmt.Println("No unit name")
        return nil
    }

    netdev, ok := networkd.GetNetDev(args[0])
    if !ok {
        return errors.New("No netdev with that name")
    }

    err := netdev.Disable()
    if err != nil {
        return err
    }

    return networkd.Restart()
}
