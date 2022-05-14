package networkd

import (
	"fmt"
	"net"
	"os/exec"
)

type Interface struct {
	Name  string
	NetIf *net.Interface
}

func (self *Interface) Delete() error {
	cmd := exec.Command("ip", "link", "show", self.Name)
	if err := cmd.Run(); err != nil {
		return nil
	}

	cmd = exec.Command("ip", "link", "del", self.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to delete link %s", self.Name)
	}

	return nil
}

func NewInterface(name string) *Interface {
	intf := Interface{Name: name}

	netif, err := net.InterfaceByName(name)
	if err == nil {
		// TODO log
		intf.NetIf = netif
	}

	return &intf
}
