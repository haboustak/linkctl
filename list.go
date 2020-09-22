package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/haboustak/linkctl/internal/networkd"
)

var (
	showAll   bool
	quietMode bool
)

var cmdList = &Command{
	Name: "list",
	Run:  list,
	Usage: `Usage:
    linkctl [-h] list [-a] [-q]

Show systemd-networkd netdev links

Options:
    -a      show all links
    -h      show this help
    -q      only print link names
`,
}

func init() {
	cmdList.Flags.BoolVar(&showAll, "a", false, "show all links")
	cmdList.Flags.BoolVar(&quietMode, "q", false, "only print link names")
}

func ansiPad(status string) string {
	return fmt.Sprintf("%s\x1B[%16dm", status, 0)
}

func ansiColorStatus(status networkd.LinkStatus) string {
	if !IsATTY {
		return string(status)
	}

	switch status {
	case networkd.LinkEnabled:
		return fmt.Sprintf("\x1B[0;1;32m%s\x1B[0000000m", status)
	case networkd.LinkUserDefined:
		return fmt.Sprintf("\x1B[0;1;38;5;185m%s\x1B[0m", status)
	default:
		return ansiPad(string(status))
	}
}

func list(self *Command) error {
	links := networkd.ListNetDev(showAll)
	if links == nil {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 16, 3, 4, ' ', tabwriter.TabIndent)
	if !quietMode {
		fmt.Fprintf(w, "NAME\tTYPE\t%s\tDESCRIPTION\t\n", ansiPad("STATUS"))
	}

	for _, link := range links {
		printLink(w, link)
	}

	w.Flush()
	return nil
}
func printLink(w io.Writer, link *networkd.NetDev) {
	if quietMode {
		fmt.Fprintln(w, link.Name)
	} else {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
			link.Name, link.Kind,
			ansiColorStatus(link.Status),
			link.Description)
	}
}
