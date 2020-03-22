// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"os"
	"path"
	"strings"

	"rsc.io/goversion/version"
)

// CmdName returns either the base name of the current binary
// using os.Executable. If os.Executable fails (it sholdn't), then
// "cmd" is returned.
func CmdName() string {
	e, err := os.Executable()
	if err != nil {
		return "cmd"
	}
	var ret string
	v, err := version.ReadExe(e)
	if err != nil {
		ret = strings.TrimSuffix(strings.ToLower(e), ".exe")
	} else {
		// v is like:
		// "path\ttailscale.com/cmd/tailscale\nmod\ttailscale.com\t(devel)\t\ndep\tgithub.com/apenwarr/fixconsole\tv0.0.0-20191012055117-5a9f6489cc29\th1:muXWUcay7DDy1/hEQWrYlBy+g0EuwT70sBHg65SeUc4=\ndep\tgithub....
		for _, line := range strings.Split(v.ModuleInfo, "\n") {
			if strings.HasPrefix(line, "path\t") {
				ret = path.Base(strings.TrimPrefix(line, "path\t"))
				break
			}
		}
	}
	if ret == "" {
		return "cmd"
	}
	return ret
}
