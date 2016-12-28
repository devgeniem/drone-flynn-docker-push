package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// default docker registry
	defaultRegistry = "https://index.docker.io/v1/"
)

type (
	// Daemon defines Docker daemon parameters.
	Daemon struct {
		Registry      string   // Docker registry
		Mirror        string   // Docker registry mirror
		Insecure      bool     // Docker daemon enable insecure registries
		StorageDriver string   // Docker daemon storage driver
		StoragePath   string   // Docker daemon storage path
		Disabled      bool     // DOcker daemon is disabled (already running)
		Debug         bool     // Docker daemon started in debug mode
		Bip           string   // Docker daemon network bridge IP address
		DNS           []string // Docker daemon dns server
		MTU           string   // Docker daemon mtu setting
		IPv6          bool     // Docker daemon IPv6 networking
	}

	// Login defines Flynn login parameters.
	Flynn struct {
		Domain   string // Flynn domain
		TLSPin   string // Flynn TLS PIN
		Key      string // Flynn Key
        App      string // Flynn application name
	}

	// Build defines Docker build parameters.
	Build struct {
		Name       string   // Docker build using default named tag
		Dockerfile string   // Docker build Dockerfile
		Context    string   // Docker build context
		Tags       []string // Docker build tags
		Args       []string // Docker build args
		Repo       string   // Docker build repository
	}

	// Plugin defines the Docker plugin parameters.
	Plugin struct {
		Flynn  Flynn  // Flynn login configuration
		Build  Build  // Docker build configuration
		Daemon Daemon // Docker daemon configuration
		Dryrun bool   // Docker push is skipped
	}
)

// Exec executes the plugin step
func (p Plugin) Exec() error {

	// TODO execute code remove dangling images
	// this is problematic because we are running docker in scratch which does
	// not have bash, so we need to hack something together
	// docker images --quiet --filter=dangling=true | xargs --no-run-if-empty docker rmi

	/*
		cmd = exec.Command("docker", "images", "-q", "-f", "dangling=true")
		cmd = exec.Command("docker", append([]string{"rmi"}, images...)...)
	*/

	// start the Docker daemon server
	if !p.Daemon.Disabled {
		cmd := commandDaemon(p.Daemon)
		if p.Daemon.Debug {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stdout = ioutil.Discard
			cmd.Stderr = ioutil.Discard
		}
		go func() {
			trace(cmd)
			cmd.Run()
		}()
	}

	// poll the docker daemon until it is started. This ensures the daemon is
	// ready to accept connections before we proceed.
	for i := 0; i < 15; i++ {
		cmd := commandDockerInfo()
		err := cmd.Run()
		if err == nil {
			break
		}
		time.Sleep(time.Second * 1)
	}

	// login to the Docker registry
	if p.Flynn.Key != ""  && p.Flynn.TLSPin != "" {
		cmd := commandLoginFlynn(p.Flynn)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Error authenticating: %s", err)
		}
	} else {
		fmt.Println("Error: Flynn credentials or TLS pin was not provided...")
        return nil
	}

	var cmds []*exec.Cmd
    cmds = append(cmds, commandFlynnVersion())       // flynn version
	cmds = append(cmds, commandDockerVersion())      // docker version
	cmds = append(cmds, commandDockerInfo())         // docker info
	cmds = append(cmds, commandBuild(p.Build)) // docker build
    cmds = append(cmds, commandPushFlynn(p.Flynn, p.Build)) // push image to Flynn

	// execute all commands in batch mode.
	for _, cmd := range cmds {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		trace(cmd)

		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

const dockerExe = "/usr/local/bin/docker"
const flynnExe = "/usr/local/bin/flynn"

// helper function to login into flynn useing the flynn client binary.
func commandLoginFlynn(flynn Flynn) *exec.Cmd {
    return exec.Command(
        flynnExe, "cluster", "add",
        "--no-git", "--docker",
        "-p", flynn.TLSPin,
        "default",
        flynn.Domain,
        flynn.Key,
    )
}

// helper function to login into flynn useing the flynn client binary.
func commandPushFlynn(flynn Flynn, build Build) *exec.Cmd {
    return exec.Command(
        flynnExe,
        "-a", flynn.App,
        "docker", "push", build.Name,
    )
}

// helper function to check the flynn client version.
func commandFlynnVersion() *exec.Cmd {
    return exec.Command(flynnExe, "version")
}

// helper function to check the docker version.
func commandDockerVersion() *exec.Cmd {
	return exec.Command(dockerExe, "version")
}

// helper function to create the docker info command.
func commandDockerInfo() *exec.Cmd {
    return exec.Command(dockerExe, "info")
}

// helper function to create the docker build command.
func commandBuild(build Build) *exec.Cmd {
	cmd := exec.Command(
		dockerExe, "build",
		"--pull=true",
		"--rm=true",
		"-f", build.Dockerfile,
		"-t", build.Name,
	)

	for _, arg := range build.Args {
		cmd.Args = append(cmd.Args, "--build-arg", arg)
	}
	cmd.Args = append(cmd.Args, build.Context)
	return cmd
}

// helper function to create the docker tag command.
func commandTag(build Build, tag string) *exec.Cmd {
	var (
		source = build.Name
		target = fmt.Sprintf("%s:%s", build.Repo, tag)
	)
	return exec.Command(
		dockerExe, "tag", source, target,
	)
}

// helper function to create the docker push command.
func commandPush(build Build, tag string) *exec.Cmd {
	target := fmt.Sprintf("%s:%s", build.Repo, tag)
	return exec.Command(dockerExe, "push", target)
}

// helper function to create the docker daemon command.
func commandDaemon(daemon Daemon) *exec.Cmd {
	args := []string{"daemon", "-g", daemon.StoragePath}

	if daemon.StorageDriver != "" {
		args = append(args, "-s", daemon.StorageDriver)
	}
	if daemon.Insecure && daemon.Registry != "" {
		args = append(args, "--insecure-registry", daemon.Registry)
	}
	if daemon.IPv6 {
		args = append(args, "--ipv6")
	}
	if len(daemon.Mirror) != 0 {
		args = append(args, "--registry-mirror", daemon.Mirror)
	}
	if len(daemon.Bip) != 0 {
		args = append(args, "--bip", daemon.Bip)
	}
	for _, dns := range daemon.DNS {
		args = append(args, "--dns", dns)
	}
	if len(daemon.MTU) != 0 {
		args = append(args, "--mtu", daemon.MTU)
	}
	return exec.Command(dockerExe, args...)
}

// trace writes each command to stdout with the command wrapped in an xml
// tag so that it can be extracted and displayed in the logs.
func trace(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
}
