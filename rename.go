package main

import (
	"errors"
	"fmt"
    "net"

	"github.com/haboustak/linkctl/internal/networkd"
)

var cmdRename = &Command{
	Name: "rename",
	Run:  rename,
    Usage: `Usage:
    linkctl [-h] rename LINK [NEWNAME]

Enable a netdev link

Arguments:
    LINK        name of the link to rename
    NEWNAME     new name for the link. If not specified the link is
                reset to its default name.

Options:
    -h          show this help
`,
}

func clearName(netdev *networkd.NetDev) error {
	return netdev.ResetName()
}

func setName(netdev *networkd.NetDev, name string) error {
	if _, ok := networkd.GetNetDev(name); ok {
		return fmt.Errorf("A link with the name %s already exists", name)
	}

    if _, err := net.InterfaceByName(name); err == nil {
		return fmt.Errorf("A link with the name %s already exists", name)
    }

	return netdev.Rename(name)
}

func rename(self *Command) error {
	args := self.Flags.Args()

	if len(args) < 1 {
		return errors.New("You must provide the name of the link to rename")
	}
	oldName := args[0]

	netdev, ok := networkd.GetNetDev(oldName)
	if !ok {
		return fmt.Errorf("No link with the name %s", oldName)
	}

	if len(args) < 2 {
		if err := clearName(netdev); err != nil {
			return err
		}
	} else if err := setName(netdev, args[1]); err != nil {
		return err
	}

	return networkd.Restart()
}
