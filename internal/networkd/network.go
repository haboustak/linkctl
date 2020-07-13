package networkd

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strings"
)

type Network struct {
    Intf    *net.Interface
    Unit    *Unit
}

var intfNetwork map[string]*Network

func init() {
    intfNetwork = make(map[string]*Network)
}

func getNetworkUnit(intf *net.Interface) (*Unit, error) {
    var networkFile string

    stateFile := fmt.Sprintf("/run/systemd/netif/links/%d", intf.Index)
    file, err := os.Open(stateFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if !strings.HasPrefix(line, "NETWORK_FILE") {
            continue
        }
        stateParts := strings.SplitN(line, "=", 2)
        if len(stateParts) != 2 {
            return nil, fmt.Errorf("Unable to parse state file for %d", intf.Index)
        }
        networkFile = stateParts[1]
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    unit, err := NewUnit(networkFile)
    return unit, err
}

func NetworkFromIntf(intfName string) (*Network, error) {
    if network, ok := intfNetwork[intfName]; ok {
        return network, nil
    }

    intf, err := net.InterfaceByName(intfName)
    if err != nil {
        return nil, fmt.Errorf("No interface %s", intfName)
    }

    networkUnit, err := getNetworkUnit(intf)
    if err != nil {
        return nil, err
    }

    intfNetwork[intfName] = &Network{Intf: intf, Unit: networkUnit}
    return intfNetwork[intfName], nil
}
