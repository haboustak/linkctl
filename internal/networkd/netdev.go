package networkd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NetDev struct {
	Name          string
	Kind          string
	Description   string
	Unit          *Unit
	Status        LinkStatus
	RenameUnit    *Unit
	Interface     *Interface
	ParentNetwork *Network
}

type LinkStatus string
type LinkType string

const (
	LinkEnabled     = "enabled"
	LinkDisabled    = "disabled"
	LinkUserDefined = "user-defined"
)

const (
	EnabledLink   LinkType = "enabled"
	AvailableLink LinkType = "available"
)

var netdevs map[string]*NetDev

func (self *NetDev) Enable() error {
	switch self.Status {
	case LinkEnabled:
		return fmt.Errorf("The link %s is already enabled", self.Name)
	case LinkUserDefined:
		return fmt.Errorf("The link %s is user-defined and cannot be enabled", self.Name)
	}

	self.Status = LinkEnabled

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	linkedName := fmt.Sprintf(
		"/etc/systemd/network/%s",
		self.Unit.Name)
	if err := os.Symlink(self.Unit.Path, linkedName); err != nil {
		return fmt.Errorf("Failed to create unit symlink %s", linkedName)
	}

	return nil
}

func (self *NetDev) Disable() error {
	switch self.Status {
	case LinkDisabled:
		return fmt.Errorf("The link %s is already disabled", self.Name)
	case LinkUserDefined:
		return fmt.Errorf("The link %s is user-defined and cannot be disabled", self.Name)
	}

	self.Status = LinkDisabled

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	err := os.Remove(self.Unit.Path)
	if err != nil {
		return err
	}

	return self.Interface.Delete()
}

func (self *NetDev) Rename(newName string) error {
	if self.RenameUnit == nil {
		renameFile := fmt.Sprintf(
			"/etc/systemd/network/%s.d/name.conf",
			self.Unit.Name)
		unit, err := NewUnit(renameFile)
		if err != nil {
			return fmt.Errorf("Failed to load unit %s: %w", renameFile, err)
		}
		self.RenameUnit = unit
	}

	self.RenameUnit.File.NewSection("NetDev")
	self.RenameUnit.File.Section("NetDev").NewKey("Name", newName)
	if err := self.RenameUnit.Save(); err != nil {
		return fmt.Errorf("Unable to save dropin unit %s: %w",
			self.RenameUnit.Path, err)
	}

	self.Interface.Delete()

	if err := self.Reload(); err != nil {
		return err
	}

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) ResetName() error {
	if self.RenameUnit == nil {
		return fmt.Errorf("The link %s has not been renamed", self.Name)
	}

	self.RenameUnit.File.Section("NetDev").DeleteKey("Name")
	if self.RenameUnit.IsEmpty() {
		if err := self.RenameUnit.Delete(); err != nil {
			return err
		}
	} else if err := self.RenameUnit.Save(); err != nil {
		return err
	}

	self.Interface.Delete()

	if err := self.Reload(); err != nil {
		return err
	}

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) Reload() error {
	applyConfig(self, self.Unit)

	if err := applyDropinConfigs(self); err != nil {
		return err
	}

	return nil
}

func parseUnitName(netdev *NetDev) (string, error) {
	nameParts := strings.SplitN(netdev.Unit.Name, "-", 2)
	if len(nameParts) != 2 {
		return "", fmt.Errorf(
			"link %s does not have a conforming unit name (%s)",
			netdev.Name, netdev.Unit.Name)
	}

	networkParts := strings.SplitN(nameParts[1], ".", 2)
	if len(networkParts) != 2 {
		return "", fmt.Errorf(
			"link %s does not have a conforming unit name (%s)",
			netdev.Name, netdev.Unit.Name)
	}
	intfName := networkParts[0]

	return intfName, nil
}

func applyConfig(netdev *NetDev, unit *Unit) {
	section := unit.File.Section("NetDev")

	var keys = []string{"Name", "Kind", "Description"}

	for _, key := range keys {
		if section.HasKey(key) {
			value := section.Key(key).String()
			switch key {
			case "Name":
				netdev.Name = value
				netdev.Interface = NewInterface(netdev.Name)
			case "Kind":
				netdev.Kind = value
			case "Description":
				netdev.Description = value
			}
		}
	}
}

func applyDropinConfigs(netdev *NetDev) error {
	dropinPath := fmt.Sprintf("/etc/systemd/network/%s.d/*.conf", netdev.Unit.Name)
	dropins, err := filepath.Glob(dropinPath)
	if err != nil {
		return fmt.Errorf("Failed to list units at %s: %w", dropinPath, err)
	}

	for _, dropin := range dropins {
		dropinUnit, err := NewUnit(dropin)
		if err != nil {
			// TODO log.warn
			continue
		}

		origName := netdev.Name
		// TODO log.warn
		applyConfig(netdev, dropinUnit)

		if netdev.Name != origName {
			netdev.RenameUnit = dropinUnit
		}
	}

	return nil
}

func NewNetDev(path string, linkType LinkType) (*NetDev, error) {
	var netdev NetDev

	unit, err := NewUnit(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load unit %s: %w", path, err)
	}
	netdev.Unit = unit

	// TODO Log parsing errors
	intfName, _ := parseUnitName(&netdev)
	netdev.ParentNetwork = NetworkFromIntf(intfName)

	netdev.Status = LinkDisabled
	if linkType == EnabledLink {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}

		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			netdev.Status = LinkEnabled
		} else {
			netdev.Status = LinkUserDefined
		}
	}

	if err := netdev.Reload(); err != nil {
		return nil, err
	}

	return &netdev, nil
}

func loadNetDevs() {
	linkTypes := [2]LinkType{EnabledLink, AvailableLink}
	configPaths := map[LinkType][]string{
		EnabledLink:   []string{"/etc/systemd/network/*.netdev"},
		AvailableLink: []string{"/etc/linkctl/user/*.netdev", "/etc/linkctl/system/*.netdev", "/etc/systemd/network/netdev.available/*.netdev"},
	}

	if netdevs != nil {
		return
	}

	netdevs = make(map[string]*NetDev)
	for _, linkType := range linkTypes {
		for _, path := range configPaths[linkType] {
			files, err := filepath.Glob(path)
			if err != nil {
				panic(err)
			}

			for _, file := range files {
				netdev, err := NewNetDev(file, linkType)
				if err != nil {
					continue
				}
				_, exist := netdevs[netdev.Name]
				if !exist {
					netdevs[netdev.Name] = netdev
				}
			}
		}
	}
}

func ListNetDev(listAll bool) []*NetDev {
	loadNetDevs()

	var keys []string
	var values []*NetDev
	for k, v := range netdevs {
		if listAll || v.Status != LinkDisabled {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		values = append(values, netdevs[k])
	}
	return values
}

func GetNetDev(linkName string) (*NetDev, bool) {
	loadNetDevs()

	link, ok := netdevs[linkName]
	return link, ok
}
