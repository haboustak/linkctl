# linkctl
A utility for managing systemd-networkd virtual interfaces.

```
$ linkctl -h
Usage:

    linkctl <command> [arguments]

The commands are:
    list        list netdev links
    enable      enable a netdev link configuration
    disable     disable a netdev link configuration
    rename      rename a netdev link
```

## Examples
List  all enabled and available netdev links.
``` bash
# linkctl list [-a]
$ linkctl list -a
NAME           TYPE           STATUS
othernet       vlan           user-defined
test.300       vlan           enabled
test.301       vlan           enabled
test.302       vlan           disabled
test.310       vlan           disabled
test.555       vlan           enabled
test.600       vlan           disabled
testroot       vlan           disabled
```

Enable a link
``` bash
# linkctl enable LINK
$ sudo linkctl enable test.300
```

Rename a link
``` bash
# linkctl rename LINK [NEWNAME]
$ sudo linkctl rename test.600 lan
```
