package main

import (
    "errors"
    "fmt"
    "github.com/haboustak/linkctl/internal/networkd"
)

var cmdRename = &Command {
    Name:   "rename",
    Run: rename,
}

var resetName bool

func init() {
}

func clearName(netdev *networkd.NetDev) error {
    return netdev.ResetName()
}

func setName(netdev *networkd.NetDev, name string) error {
    _, ok := networkd.GetNetDev(name)
    if ok {
        return fmt.Errorf("A link with the name %s already exists", name)
    }

    return netdev.Rename(name)
}

func rename(self *Command) error {
    args := self.Flags.Args()

    if len(args) < 1 {
        return errors.New("No unit name")
    }
    oldName := args[0]

    netdev, ok := networkd.GetNetDev(oldName)
    if !ok {
        return fmt.Errorf("No link with the name %s", oldName)
    }

    var err error
    if len(args) < 2 {
        err = clearName(netdev)
    } else {
        err = setName(netdev, args[1])
    }

    if err != nil {
        return err
    }

    return networkd.Restart()
}

