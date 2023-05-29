package main

import (
	"fmt"
	"log"

	"github.com/vishvananda/netlink"
)

func isDefaultRoute(route netlink.Route) bool {
	if route.Dst == nil {
		return true
	}
	mask := route.Dst.Mask
	for _, b := range mask {
		if b != 0 {
			return false
		}
	}
	return true
}

func isULAGW(route netlink.Route) bool {
	if route.Gw == nil {
		return false
	}
	return route.Gw[0] == 0xFD
}

func isLANv4Route(route netlink.Route) bool {
	if route.Dst == nil {
		return false
	}
	if route.Dst.IP[0] != 10 {
		return false
	}
	return route.Dst.Mask[0] == 0xFF && route.Dst.Mask[1] == 0xFF && route.Dst.Mask[2] == 0x00 && route.Dst.Mask[3] == 0x00
}

func netcheck(id string) error {
	log.Printf("[%s] Netcheck starting", id)
	links, err := netlink.LinkList()
	if err != nil {
		return err
	}

	var routeLANv4 *netlink.Route
	var routeDefaultv4 *netlink.Route
	var routeDefaultULAv6 *netlink.Route

	for _, link := range links {
		routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		for _, route := range routes {
			if isLANv4Route(route) {
				routeLANv4 = &route
				continue
			}
			if isDefaultRoute(route) {
				routeDefaultv4 = &route
				continue
			}
		}

		routes, err = netlink.RouteList(link, netlink.FAMILY_V6)
		if err != nil {
			return err
		}
		for _, route := range routes {
			if !isDefaultRoute(route) {
				continue
			}
			if !isULAGW(route) {
				continue
			}
			routeDefaultULAv6 = &route
		}
	}

	log.Printf("[%s] Routes: LANv4=%v Defaultv4=%v DefaultULAv6=%v", id, routeLANv4, routeDefaultv4, routeDefaultULAv6)
	if routeLANv4 == nil {
		log.Printf("[%s] No LANv4 route found, exiting netcheck", id)
		return nil
	}

	routeLANGWv4 := routeLANv4.Dst.IP
	routeLANGWv4[3] = 1
	log.Printf("[%s] GW IP: LANv4=%v", id, routeLANGWv4)

	if routeDefaultULAv6 != nil {
		log.Printf("[%s] Deleting DefaultULAv6 route %v", id, routeDefaultULAv6)
		err = netlink.RouteDel(routeDefaultULAv6)
		if err != nil {
			return err
		}
	}

	if routeDefaultv4 == nil || !routeLANGWv4.Equal(routeDefaultv4.Gw) {
		log.Printf("[%s] Changing Defaultv4 gateway to LANv4", id)
		if routeDefaultv4 != nil {
			err = netlink.RouteDel(routeDefaultv4)
			if err != nil {
				return fmt.Errorf("could not delete Defaultv4 %w", err)
			}
		}
		routeNewDefaultV4 := &netlink.Route{
			Gw:    routeLANGWv4,
			Table: routeLANv4.Table,
		}
		err = netlink.RouteAdd(routeNewDefaultV4)
		if err != nil {
			return fmt.Errorf("could not add NewDefaultv4 %w", err)
		}
	}

	log.Printf("[%s] Netcheck finished successfully", id)

	return nil
}
