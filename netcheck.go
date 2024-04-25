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
		return fmt.Errorf("could not LinkList %w", err)
	}

	routesLANv4 := make([]netlink.Route, 0)
	routesDefaultv4 := make([]netlink.Route, 0)
	routesDefaultULAv6 := make([]netlink.Route, 0)

	for _, link := range links {
		routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
		if err != nil {
			return fmt.Errorf("could not RouteList v4 %w", err)
		}
		for _, route := range routes {
			if isLANv4Route(route) {
				routesLANv4 = append(routesLANv4, route)
				continue
			}
			if isDefaultRoute(route) {
				routesDefaultv4 = append(routesDefaultv4, route)
				continue
			}
		}

		routes, err = netlink.RouteList(link, netlink.FAMILY_V6)
		if err != nil {
			return fmt.Errorf("could not RouteList v6 %w", err)
		}
		for _, route := range routes {
			if !isDefaultRoute(route) {
				continue
			}
			if !isULAGW(route) {
				continue
			}
			routesDefaultULAv6 = append(routesDefaultULAv6, route)
		}
	}

	log.Printf("[%s] Routes: LANv4=%v Defaultv4=%v DefaultULAv6=%v", id, routesLANv4, routesDefaultv4, routesDefaultULAv6)
	if len(routesLANv4) == 0 {
		log.Printf("[%s] No LANv4 route found, exiting netcheck", id)
		return nil
	}

	var routeLANv4 netlink.Route
	routeLANv4.MTU = -1
	for _, route := range routesLANv4 {
		if routeLANv4.MTU < 0 || routeLANv4.LinkIndex > route.MTU {
			routeLANv4 = route
		}
	}

	routeLANGWv4 := routeLANv4.Dst.IP
	routeLANGWv4[3] = 1
	log.Printf("[%s] GW IP: LANv4=%v", id, routeLANGWv4)

	for _, route := range routesDefaultULAv6 {
		log.Printf("[%s] Deleting DefaultULAv6 route %v", id, route)
		err = netlink.RouteDel(&route)
		if err != nil {
			return fmt.Errorf("could not delete DefaultULAv6 %w", err)
		}
	}

	if len(routesDefaultv4) != 1 || !routeLANGWv4.Equal(routesDefaultv4[0].Gw) {
		log.Printf("[%s] Changing Defaultv4 gateway to LANv4", id)
		for _, route := range routesDefaultv4 {
			err = netlink.RouteDel(&route)
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
