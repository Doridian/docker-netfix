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

func isDockerGW(route netlink.Route) bool {
	if route.Gw == nil {
		return false
	}
	if route.Family == netlink.FAMILY_V4 {
		return route.Gw[0] == 172 && route.Gw[1] >= 16 && route.Gw[1] <= 31
	}
	if route.Family == netlink.FAMILY_V6 {
		return route.Gw[0] == 0xFD && route.Gw[1] == 0xD8 && route.Gw[2] == 0x23 && route.Gw[3] == 0x57
	}
	return false
}

func isBetterRoute(routeBase, routeQuery netlink.Route) bool {
	if routeBase.Gw == nil {
		return true
	}

	if isDockerGW(routeQuery) {
		if !isDockerGW(routeBase) {
			return false
		}
	} else if isDockerGW(routeBase) {
		return true
	}

	if routeBase.LinkIndex < routeQuery.LinkIndex {
		return true
	}

	return false
}

func netcheck(id string) error {
	log.Printf("[%s] Netcheck starting", id)
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("could not LinkList %w", err)
	}

	routesDefaultV4 := make([]netlink.Route, 0)
	routesDefaultV6 := make([]netlink.Route, 0)
	var routeTargetDefaultV4 netlink.Route
	routeTargetDefaultV4.Family = netlink.FAMILY_V4
	var routeTargetDefaultV6 netlink.Route
	routeTargetDefaultV6.Family = netlink.FAMILY_V6

	for _, link := range links {
		routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
		if err != nil {
			return fmt.Errorf("could not RouteList v4 %w", err)
		}
		for _, route := range routes {
			if !isDefaultRoute(route) {
				continue
			}

			routesDefaultV4 = append(routesDefaultV4, route)
			if isBetterRoute(routeTargetDefaultV4, route) {
				routeTargetDefaultV4 = route
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

			routesDefaultV6 = append(routesDefaultV6, route)
			if isBetterRoute(routeTargetDefaultV6, route) {
				routeTargetDefaultV6 = route
			}
		}
	}

	log.Printf("[%s] Routes: DefaultV4=%v DefaultV6=%v", id, routesDefaultV4, routesDefaultV6)

	if len(routesDefaultV4) > 0 {
		for _, route := range routesDefaultV4 {
			if routeTargetDefaultV4.Gw.Equal(route.Gw) {
				continue
			}
			log.Printf("[%s] Deleting DefaultV4=%v", id, route.Gw)
			err = netlink.RouteDel(&route)
			if err != nil {
				return fmt.Errorf("could not delete DefaultV4 %w", err)
			}
		}
	}

	if len(routesDefaultV6) > 0 {
		for _, route := range routesDefaultV6 {
			if routeTargetDefaultV6.Gw.Equal(route.Gw) {
				continue
			}
			log.Printf("[%s] Deleting DefaultV6=%v", id, route.Gw)
			err = netlink.RouteDel(&route)
			if err != nil {
				return fmt.Errorf("could not delete DefaultV6 %w", err)
			}
		}
	}

	log.Printf("[%s] Netcheck finished successfully", id)

	return nil
}
