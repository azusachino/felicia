//go:build unix

package runcv1

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func fun() {
	cmd := exec.Command("sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}

	// Set up the namespace for hostname
	cmd.SysProcAttr.UtsNamespace = &syscall.UtsNamespace{
		Host: "runcv1_hostname",
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
