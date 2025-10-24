package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/ttacon/chalk"
)

// Utility functions copied from nanovms' ops sources

func getCurrentVersion(ctx context.Context, useLatestKernel bool, arch string) (string, error) {
	var err error

	local, remote := lepton.LocalReleaseVersion, lepton.LatestReleaseVersion
	if local == "0.0" || (useLatestKernel && remote != local) {
		if runtime.GOARCH != arch {
			p.GetLogger(ctx).Warningf("Detected %s architecture in Elf binary, but running on %s, downloading kernel for %s", arch, runtime.GOARCH, arch)
		}
		err = lepton.DownloadReleaseImages(remote, arch)
		if err != nil {
			return "", err
		}
		return remote, nil
	}

	if parseVersion(local, 4) != parseVersion(remote, 4) {
		fmt.Println(chalk.Red, "You are running an older version of Ops.", chalk.Reset)
		fmt.Println(chalk.Red, "Update: Run", chalk.Reset, chalk.Bold.TextStyle("`ops update`"))
	}

	return local, nil
}

func parseVersion(s string, width int) int64 {
	strList := strings.Split(s, ".")
	format := fmt.Sprintf("%%s%%0%ds", width)
	v := ""
	for _, value := range strList {
		v = fmt.Sprintf(format, v, value)
	}
	var result int64
	var err error
	if result, err = strconv.ParseInt(v, 10, 64); err != nil {
		fmt.Printf("Failed to parse version %s, error is: %s", v, err)
		os.Exit(1)
	}
	return result
}

func getKernelVersion(version string) string {
	return path.Join(lepton.GetOpsHome(), version, "kernel.img")
}

// b7 = 183 = arm; 3e = 62 = x86
func archCheck(imgpath string) string {
	f, err := os.Open(imgpath)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	h := make([]byte, 19)
	_, err = f.Read(h)

	if h[18] == 183 {
		return "arm"
	}

	return "amd64"
}
