package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/vishvananda/netns"

	// "github.com/containernetworking/cni/pkg/types/current"
	"github.com/vishvananda/netlink"
)

// Assumption made is that the bridge and the ns exist beforehand

const (
	about = "plugin made by shubham"
	mtu   = 1500
)

type NetConf struct {
	types.NetConf
	BrName string `json:"bridge"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, about)
}

func cmdAdd(args *skel.CmdArgs) error {
	fmt.Println("Executing cmdAdd")

	plugin := NetConf{}
	networkNS := args.Netns

	if err := json.Unmarshal(args.StdinData, &plugin); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	// Find the name of the Bridge
	br, err := netlink.LinkByName(plugin.BrName)
	if err != nil {
		return fmt.Errorf("failed to get %q: %v", plugin.BrName, err)
	}

	// We need to create eth peers i.e. for the host and container side
	err = setupVeth(args.ContainerID, br, networkNS)
	if err != nil {
		return err
	}

	//return types.PrintResult(result, plugin.CNIVersion)
	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	fmt.Println("Executing cmdDel")
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	fmt.Println("Executing cmdCheck")
	return nil
}

func setupVeth(containerID string, br netlink.Link, networkNS string) error {
	hostVethName := "veth" + containerID[:5]
	containerVethName := "eth0"

	hostVeth, containerVeth, err := createVethPair(hostVethName, containerVethName, mtu)
	if err != nil {
		return fmt.Errorf("failed to create veth: %v", err)
	}

	// Link UP It would help to activate the interface may also assign the ip to the interface
	if err := netlink.LinkSetUp(hostVeth); err != nil {
		return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
	}

	// Open the new network namespace
	newns, err := netns.GetFromName(networkNS)
	if err != nil {
		fmt.Printf("Failed to open namespace %s: %v", networkNS, err)
		return err
	}
	defer newns.Close()

	// Save the current network namespace
	origns, err := netns.Get()
	if err != nil {
		fmt.Printf("Failed to get current namespace: %v", err)
		return err
	}
	defer origns.Close()
	// Switch to the new network namespace
	err = netns.Set(newns)
	if err != nil {
		fmt.Printf("Failed to switch to namespace %s: %v", networkNS, err)
		return err
	}

	// Create a netlink handle scoped to the new network namespace
	handle, err := netlink.NewHandle()
	if err != nil {
		fmt.Printf("Failed to create new netlink handle: %v", err)
		return err
	}
	defer handle.Delete()

	// Link UP It would help to activate the interface may also assign the ip to the interface
	if err := netlink.LinkSetUp(containerVeth); err != nil {
		return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
	}

	// Switch back to the original network namespace
	err = netns.Set(origns)
	if err != nil {
		fmt.Printf("Failed to switch back to original namespace: %v", err)
		return err
	}

	if err := netlink.LinkSetMaster(hostVeth, br.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", hostVethName, br.Attrs().Name, err)
	}

	if err := netlink.LinkSetMaster(containerVeth, br.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", containerVethName, br.Attrs().Name, err)
	}

	return nil
}

func createVethPair(hostVethName string, containerVethName string, mtu int) (*netlink.Veth, *netlink.Veth, error) {
	link := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  hostVethName,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: containerVethName,
	}

	if err := netlink.LinkAdd(link); err != nil {
		return nil, nil, fmt.Errorf("failed to create veth: %v", err)
	}

	// Find the host veth
	host, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find link named %q: %v", hostVethName, err)
	}

	veth := host.(*netlink.Veth)

	// Find the container veth
	peer, err := netlink.LinkByName(containerVethName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find link named %q: %v", containerVethName, err)
	}

	peerVeth := peer.(*netlink.Veth)

	return veth, peerVeth, nil
}
