// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"os/exec"
)

// {"version":"","id":"96c1870d5c21c324db0a5b350f5a8f5571cf514205b6d3647b893b6580a05018","pid":27146,"root":"/opt/docker/devicemapper/mnt/a45fd0cb88f52620f94215e8e19b806f787f0c649be8e5c13b737dc2d8278daf/rootfs"}
type State struct {
	Version    string `json:"version"`
	ID         string `json:"id"`
	Pid        int    `json:"pid"`
	Root       string `json:"root"`
	BundlePath string `json:"bundlePath"`
}

type Process struct {
	Env []string `json:"env"`
}

func allowKvm(state State) {
	// TODO: Use state.Pid and /proc/$pid/cgroup to determine the right devices.allow file
	path := fmt.Sprintf("/sys/fs/cgroup/devices/system.slice/docker-%s.scope/devices.allow", state.ID)
	allow, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Failed to open file: %s: %v", path, err.Error())
		return
	}

	_, err = allow.WriteString("c 10:232 rwm")
	if err != nil {
		log.Printf("Failed to write to group file: %s: %v", path, err.Error())
		return
	}

	path = fmt.Sprintf("%s/dev/kvm", state.Root)
	cmd := exec.Command("/usr/bin/mknod", path, "c", "10", "232")
	_, err = cmd.Output()
	if err != nil {
		log.Printf("Failed to run mknod: %s: %v", path, err.Error())
		return
	}

	log.Printf("Allowed /dev/kvm in new container: %s %s", path, state.ID)
}

func main() {
	var state State

	logwriter, err := syslog.New(syslog.LOG_NOTICE, "oci-kvm-hook")
	if err == nil {
		log.SetOutput(logwriter)
	}

	if err := json.NewDecoder(os.Stdin).Decode(&state); err != nil {
		log.Fatalf("Invalid json passed to OCI hook: %v", err.Error())
	}

	command := map[bool]string{true: "prestart", false: "poststop"}[state.Pid > 0]
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if command == "prestart" {
		allowKvm(state)
	}
}
