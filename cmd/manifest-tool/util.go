package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/docker/distribution/reference"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultHostname is the default built-in registry (DockerHub)
	DefaultHostname = "docker.io"
	// LegacyDefaultHostname is the old hostname used for DockerHub
	LegacyDefaultHostname = "index.docker.io"
	// DefaultRepoPrefix is the prefix used for official images in DockerHub
	DefaultRepoPrefix = "library/"
)

// list of valid os/arch values (see "Optional Environment Variables" section
// of https://golang.org/doc/install/source
var validOSArch = map[string]bool{
	"darwin/386":      true,
	"darwin/amd64":    true,
	"darwin/arm":      true,
	"darwin/arm64":    true,
	"dragonfly/amd64": true,
	"freebsd/386":     true,
	"freebsd/amd64":   true,
	"freebsd/arm":     true,
	"linux/386":       true,
	"linux/amd64":     true,
	"linux/arm":       true,
	"linux/arm/v5":    true,
	"linux/arm/v6":    true,
	"linux/arm/v7":    true,
	"linux/arm64":     true,
	"linux/arm64/v8":  true,
	"linux/ppc64":     true,
	"linux/ppc64le":   true,
	"linux/mips64":    true,
	"linux/mips64le":  true,
	"linux/s390x":     true,
	"netbsd/386":      true,
	"netbsd/amd64":    true,
	"netbsd/arm":      true,
	"openbsd/386":     true,
	"openbsd/amd64":   true,
	"openbsd/arm":     true,
	"plan9/386":       true,
	"plan9/amd64":     true,
	"solaris/amd64":   true,
	"windows/386":     true,
	"windows/amd64":   true,
	"windows/arm":     true,
}

func parseName(name string) (reference.Named, error) {
	distref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, err
	}
	hostname, remoteName := splitHostname(distref.String())
	if hostname == "" {
		return nil, fmt.Errorf("Please use a fully qualified repository name")
	}
	return reference.ParseNormalizedNamed(fmt.Sprintf("%s/%s", hostname, remoteName))
}

// splitHostname splits a repository name to hostname and remotename string.
// If no valid hostname is found, the default hostname is used. Repository name
// needs to be already validated before.
func splitHostname(name string) (hostname, remoteName string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost") {
		hostname, remoteName = DefaultHostname, name
	} else {
		hostname, remoteName = name[:i], name[i+1:]
	}
	if hostname == LegacyDefaultHostname {
		hostname = DefaultHostname
	}
	if hostname == DefaultHostname && !strings.ContainsRune(remoteName, '/') {
		remoteName = DefaultRepoPrefix + remoteName
	}
	return
}

func newResolver(username, password string, configs ...string) remotes.Resolver {
	if username != "" || password != "" {
		return docker.NewResolver(docker.ResolverOptions{
			Credentials: func(hostName string) (string, string, error) {
				return username, password, nil
			},
		})
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		logrus.Warnf("Error loading auth file: %v", err)
	}
	resolver, err := cli.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		logrus.Warnf("Error loading resolver: %v", err)
		resolver = docker.NewResolver(docker.ResolverOptions{})
	}
	return resolver
}

func isValidOSArch(os string, arch string, variant string) bool {
	osarch := fmt.Sprintf("%s/%s", os, arch)

	if variant != "" {
		osarch = fmt.Sprintf("%s/%s/%s", os, arch, variant)
	}

	_, ok := validOSArch[osarch]
	return ok
}
