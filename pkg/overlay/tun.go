package overlay

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type OverlayManager struct {
	InterfaceName string
}

func NewOverlayManager(ifName string) *OverlayManager {
	return &OverlayManager{InterfaceName: ifName}
}

func (m *OverlayManager) EnsureState(subnet string) error {
	link, err := netlink.LinkByName(m.InterfaceName)
	if err != nil {
		return m.createInterface(subnet)
	}
	return m.reconcileInterfaceState(link, subnet)
}

func (m *OverlayManager) createInterface(subnet string) error {
	la := netlink.NewLinkAttrs()
	la.Name = m.InterfaceName
	
	tun := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TUN,
	}

	if err := netlink.LinkAdd(tun); err != nil {
		return fmt.Errorf("fallo al declarar el enlace TUN: %v", err)
	}

	return m.reconcileInterfaceState(tun, subnet)
}

func (m *OverlayManager) reconcileInterfaceState(link netlink.Link, subnet string) error {
	// Asegurar que la interfaz esté arriba (UP)
	if link.Attrs().Flags&net.FlagUp == 0 {
		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("fallo al activar el enlace: %v", err)
		}
	}

	// Parsear la intención de subred
	ip, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	ipNet.IP = ip

	// Verificar si la IP ya está asignada en la interfaz
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return err
	}

	hasIP := false
	for _, addr := range addrs {
		if addr.IP.Equal(ip) {
			hasIP = true
			break
		}
	}

	// Si la intención no coincide con el estado real, aplicamos la transición
	if !hasIP {
		addr := &netlink.Addr{IPNet: ipNet}
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("fallo al asignar la subred IPv6 overlay: %v", err)
		}
		fmt.Printf("[edgelink-overlay] Interfaz %s convergida con la subred %s\n", m.InterfaceName, subnet)
	}

	return nil
}
