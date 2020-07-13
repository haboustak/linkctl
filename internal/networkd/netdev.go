package networkd

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
)

type NetDev struct {
    Name        string
    Kind        string
    Unit        *Unit
    Status      LinkStatus
    RenameUnit  *Unit
    ParentNetwork *Network
    ParentDropin *Unit
}

type LinkStatus string
type LinkType string

const (
    LinkEnabled = "enabled"
    LinkDisabled = "disabled"
    LinkUserDefined = "user-defined"
)

const (
    EnabledLink LinkType = "/etc/systemd/network/*.netdev"
    AvailableLink LinkType = "/etc/systemd/network/netdev.available/*.netdev"
)

var netdevs map[string]*NetDev

func (self *NetDev) Enable() error {
    switch self.Status {
    case LinkEnabled:
        return fmt.Errorf("The link %s is already enabled", self.Name)
    case LinkUserDefined:
        return fmt.Errorf("The link %s is user-defined and cannot be enabled", self.Name)
    }

    if self.ParentDropin == nil {
        return fmt.Errorf(
            "The parent network for %s could not be determined",
            self.Name)
    }

    self.ParentDropin.File.NewSection("Network")
    self.ParentDropin.File.Section("Network").NewKey("VLAN", self.Name)
    self.ParentDropin.Save()

    linkedName := fmt.Sprintf(
        "/etc/systemd/network/%s",
        self.Unit.Name)
    return os.Symlink(self.Unit.Path, linkedName)
}

func (self *NetDev) Disable() error {
    switch self.Status {
    case LinkDisabled:
        return fmt.Errorf("The link %s is already disabled", self.Name)
    case LinkUserDefined:
        return fmt.Errorf("The link %s is user-defined and cannot be disabled", self.Name)
    }

    if self.ParentDropin != nil {
        self.ParentDropin.Delete()
    }

    err := os.Remove(self.Unit.Path)
    if err != nil {
        return err
    }

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

func (self *NetDev) Rename(newName string) error {
    if self.RenameUnit == nil {
        renameFile := fmt.Sprintf(
            "/etc/systemd/network/%s.d/name.conf",
            self.Unit.Name)
        unit, err := NewUnit(renameFile)
        if err != nil {
            return err
        }
        self.RenameUnit = unit
    }

    self.RenameUnit.File.NewSection("NetDev")
    self.RenameUnit.File.Section("NetDev").NewKey("Name", newName)

    return self.RenameUnit.Save()
}

func (self *NetDev) ResetName() error {
    if self.RenameUnit == nil {
        return fmt.Errorf("The link %s has not been renamed", self.Name)
    }

    self.RenameUnit.File.Section("NetDev").DeleteKey("Name")

    if self.RenameUnit.IsEmpty() {
        return self.RenameUnit.Delete()
    }

    return self.RenameUnit.Save()
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

    var keys = []string{"Name", "Kind"}

    for _, key := range keys {
        if section.HasKey(key) {
            value := section.Key(key).String()
            switch key {
            case "Name":
                netdev.Name = value
            case "Kind":
                netdev.Kind = value
            }
        }
    }
}

func applyDropinConfigs(netdev *NetDev) {
    dropinPath := fmt.Sprintf("/etc/systemd/network/%s.d/*.conf", netdev.Unit.Name)
    dropins, err := filepath.Glob(dropinPath)
    if err != nil {
        return
    }

    for _, dropin := range dropins {
        dropinUnit, err := NewUnit(dropin)
        if err != nil {
            continue
        }

        origName := netdev.Name
        applyConfig(netdev, dropinUnit)
        if origName != "" && netdev.Name != origName {
            netdev.RenameUnit = dropinUnit
        }
    }
}

func NewNetDev(path string, linkType LinkType) (*NetDev, error) {
    var netdev NetDev

    unit, err := NewUnit(path)
    if err != nil {
        return nil, err
    }
    netdev.Unit = unit

    if intf, err := parseUnitName(&netdev); err == nil {
        if network, err := NetworkFromIntf(intf); err == nil {
            netdev.ParentNetwork = network
            dropinPath := fmt.Sprintf(
                "/etc/systemd/network/%s.d/%s.conf",
                netdev.ParentNetwork.Unit.Name,
                strings.TrimSuffix(netdev.Unit.Name, ".netdev"))
            if dropinUnit, err := NewUnit(dropinPath); err == nil {
                netdev.ParentDropin = dropinUnit
            }
        }
    }

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

    applyConfig(&netdev, netdev.Unit)

    applyDropinConfigs(&netdev)

    return &netdev, nil
}

func loadNetDevs() {
    linkTypes := [2]LinkType{EnabledLink, AvailableLink}

    if netdevs != nil {
        return
    }

    netdevs = make(map[string]*NetDev)
    for _, linkType := range linkTypes {
        files, err := filepath.Glob(string(linkType))
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

    link,ok := netdevs[linkName]
    return link, ok
}
