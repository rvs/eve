// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wgengine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/tailscale/wireguard-go/device"
	"github.com/tailscale/wireguard-go/tun"
	"github.com/tailscale/wireguard-go/wgcfg"
	"tailscale.com/atomicfile"
	"tailscale.com/types/logger"
)

const DefaultTunName = "tailscale0"

type linuxRouter struct {
	logf    func(fmt string, args ...interface{})
	tunname string
	local   wgcfg.CIDR
	routes  map[wgcfg.CIDR]struct{}

	ipt4 *iptables.IPTables
}

func newUserspaceRouter(logf logger.Logf, _ *device.Device, tunDev tun.Device) (Router, error) {
	tunname, err := tunDev.Name()
	if err != nil {
		return nil, err
	}

	ipt4, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return nil, err
	}

	return &linuxRouter{
		logf:    logf,
		tunname: tunname,
		ipt4:    ipt4,
	}, nil
}

func cmd(args ...string) *exec.Cmd {
	if len(args) == 0 {
		log.Fatalf("exec.Cmd(%#v) invalid; need argv[0]\n", args)
	}
	return exec.Command(args[0], args[1:]...)
}

func (r *linuxRouter) Up() error {
	out, err := cmd("ip", "link", "set", r.tunname, "up").CombinedOutput()
	if err != nil {
		// TODO: this should return an error; why is it calling log.Fatalf?
		// Audit callers to make sure they're handling errors.
		log.Fatalf("running ip link failed: %v\n%s", err, out)
	}

	err = r.ipt4.AppendUnique("filter", "FORWARD", r.forwardRule()...)
	if err != nil {
		r.logf("iptables forward failed: %v", err)
	}
	err = r.ipt4.AppendUnique("nat", "POSTROUTING", r.natRule()...)
	if err != nil {
		r.logf("iptables nat failed: %v", err)
	}
	return nil
}

func (r *linuxRouter) SetRoutes(rs RouteSettings) error {
	var errq error

	if rs.LocalAddr != r.local {
		if r.local != (wgcfg.CIDR{}) {
			addrdel := []string{"ip", "addr",
				"del", r.local.String(),
				"dev", r.tunname}
			out, err := cmd(addrdel...).CombinedOutput()
			if err != nil {
				r.logf("addr del failed: %v: %v\n%s", addrdel, err, out)
				if errq == nil {
					errq = err
				}
			}
		}
		addradd := []string{"ip", "addr",
			"add", rs.LocalAddr.String(),
			"dev", r.tunname}
		out, err := cmd(addradd...).CombinedOutput()
		if err != nil {
			r.logf("addr add failed: %v: %v\n%s", addradd, err, out)
			if errq == nil {
				errq = err
			}
		}
	}

	newRoutes := make(map[wgcfg.CIDR]struct{})
	for _, peer := range rs.Cfg.Peers {
		for _, route := range peer.AllowedIPs {
			newRoutes[route] = struct{}{}
		}
	}
	for route := range r.routes {
		if _, keep := newRoutes[route]; !keep {
			net := route.IPNet()
			nip := net.IP.Mask(net.Mask)
			nstr := fmt.Sprintf("%v/%d", nip, route.Mask)
			addrdel := []string{"ip", "route",
				"del", nstr,
				"via", r.local.IP.String(),
				"dev", r.tunname}
			out, err := cmd(addrdel...).CombinedOutput()
			if err != nil {
				r.logf("addr del failed: %v: %v\n%s", addrdel, err, out)
				if errq == nil {
					errq = err
				}
			}
		}
	}
	for route := range newRoutes {
		if _, exists := r.routes[route]; !exists {
			net := route.IPNet()
			nip := net.IP.Mask(net.Mask)
			nstr := fmt.Sprintf("%v/%d", nip, route.Mask)
			addradd := []string{"ip", "route",
				"add", nstr,
				"via", rs.LocalAddr.IP.String(),
				"dev", r.tunname}
			out, err := cmd(addradd...).CombinedOutput()
			if err != nil {
				r.logf("addr add failed: %v: %v\n%s", addradd, err, out)
				if errq == nil {
					errq = err
				}
			}
		}
	}

	r.local = rs.LocalAddr
	r.routes = newRoutes

	// TODO: this:
	if false {
		if err := r.replaceResolvConf(rs.DNS, rs.DNSDomains); err != nil {
			errq = fmt.Errorf("replacing resolv.conf failed: %v", err)
		}
	}
	return errq
}

func (r *linuxRouter) forwardRule() []string {
	return []string{
		"-m", "comment", "--comment", "tailscale",
		"-i", r.tunname,
		"-j", "ACCEPT",
	}
}

func (r *linuxRouter) natRule() []string {
	// TODO(apenwarr): hardcoded eth0 interface is obviously not right.
	return []string{
		"-m", "comment", "--comment", "tailscale",
		"-o", "eth0",
		"-j", "MASQUERADE",
	}
}

func (r *linuxRouter) Close() error {
	var ret error
	set := func(err error) {
		if ret == nil && err != nil {
			ret = err
		}
	}
	if err := r.restoreResolvConf(); err != nil {
		r.logf("failed to restore system resolv.conf: %v", err)
		set(err)
	}
	set(r.ipt4.Delete("filter", "FORWARD", r.forwardRule()...))
	set(r.ipt4.Delete("nat", "POSTROUTING", r.natRule()...))

	// TODO(apenwarr): clean up routes etc.

	return ret
}

const (
	tsConf     = "/etc/resolv.tailscale.conf"
	backupConf = "/etc/resolv.pre-tailscale-backup.conf"
	resolvConf = "/etc/resolv.conf"
)

func (r *linuxRouter) replaceResolvConf(servers []wgcfg.IP, domains []string) error {
	if len(servers) == 0 {
		return r.restoreResolvConf()
	}

	// First write the tsConf file.
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "# resolv.conf(5) file generated by tailscale\n")
	fmt.Fprintf(buf, "#     DO NOT EDIT THIS FILE BY HAND -- CHANGES WILL BE OVERWRITTEN\n\n")
	for _, ns := range servers {
		fmt.Fprintf(buf, "nameserver %s\n", ns)
	}
	if len(domains) > 0 {
		fmt.Fprintf(buf, "search "+strings.Join(domains, " ")+"\n")
	}
	f, err := ioutil.TempFile(filepath.Dir(tsConf), filepath.Base(tsConf)+".*")
	if err != nil {
		return err
	}
	f.Close()
	if err := atomicfile.WriteFile(f.Name(), buf.Bytes(), 0644); err != nil {
		return err
	}
	os.Chmod(f.Name(), 0644) // ioutil.TempFile creates the file with 0600
	if err := os.Rename(f.Name(), tsConf); err != nil {
		return err
	}

	if linkPath, err := os.Readlink(resolvConf); err != nil {
		// Remove any old backup that may exist.
		os.Remove(backupConf)

		// Backup the existing /etc/resolv.conf file.
		contents, err := ioutil.ReadFile(resolvConf)
		if os.IsNotExist(err) {
			// No existing /etc/resolv.conf file to backup.
			// Nothing to do.
			return nil
		} else if err != nil {
			return err
		}
		if err := atomicfile.WriteFile(backupConf, contents, 0644); err != nil {
			return err
		}
	} else if linkPath != tsConf {
		// Backup the existing symlink.
		os.Remove(backupConf)
		if err := os.Symlink(linkPath, backupConf); err != nil {
			return err
		}
	} else {
		// Nothing to do, resolvConf already points to tsConf.
		return nil
	}

	os.Remove(resolvConf)
	if err := os.Symlink(tsConf, resolvConf); err != nil {
		return nil
	}

	out, _ := exec.Command("service", "systemd-resolved", "restart").CombinedOutput()
	if len(out) > 0 {
		r.logf("service systemd-resolved restart: %s", out)
	}
	return nil
}

func (r *linuxRouter) restoreResolvConf() error {
	if _, err := os.Stat(backupConf); err != nil {
		if os.IsNotExist(err) {
			return nil // no backup resolv.conf to restore
		}
		return err
	}
	if ln, err := os.Readlink(resolvConf); err != nil {
		return err
	} else if ln != tsConf {
		return fmt.Errorf("resolv.conf is not a symlink to %s", tsConf)
	}
	if err := os.Rename(backupConf, resolvConf); err != nil {
		return err
	}
	os.Remove(tsConf) // best effort removal of tsConf file
	out, _ := exec.Command("service", "systemd-resolved", "restart").CombinedOutput()
	if len(out) > 0 {
		r.logf("service systemd-resolved restart: %s", out)
	}
	return nil
}
