package version

import (
	"fmt"
	"runtime"
)

type Version struct {
	Version      string `json:"version" protobuf:"bytes,1,opt,name=version"`
	BuildDate    string `json:"buildDate" protobuf:"bytes,2,opt,name=buildDate"`
	GitCommit    string `json:"gitCommit" protobuf:"bytes,3,opt,name=gitCommit"`
	GitTag       string `json:"gitTag" protobuf:"bytes,4,opt,name=gitTag"`
	GitTreeState string `json:"gitTreeState" protobuf:"bytes,5,opt,name=gitTreeState"`
	GoVersion    string `json:"goVersion" protobuf:"bytes,6,opt,name=goVersion"`
	Compiler     string `json:"compiler" protobuf:"bytes,7,opt,name=compiler"`
	Platform     string `json:"platform" protobuf:"bytes,8,opt,name=platform"`
}

var (
	version      = "v0.0.0"               // value from VERSION file
	buildDate    = "1970-01-01T00:00:00Z" // output from `date -u +'%Y-%m-%dT%H:%M:%SZ'`
	gitCommit    = ""                     // output from `git rev-parse HEAD`
	gitTag       = ""                     // output from `git describe --exact-match --tags HEAD` (if clean tree state)
	gitTreeState = ""                     // determined from `git status --porcelain`. either 'clean' or 'dirty'
)

func GetVersion() Version {
	var versionStr string
	if gitCommit != "" && gitTag != "" && gitTreeState == "clean" {
		// if we have a clean tree state and the current commit is tagged,
		// this is an official release.
		versionStr = gitTag
	} else {
		// otherwise formulate a version string based on as much metadata
		// information we have available.
		versionStr = version
		if len(gitCommit) >= 7 {
			versionStr += "+" + gitCommit[0:7]
			if gitTreeState != "clean" {
				versionStr += ".dirty"
			}
		} else {
			versionStr += "+unknown"
		}
	}
	return Version{
		Version:      versionStr,
		BuildDate:    buildDate,
		GitCommit:    gitCommit,
		GitTag:       gitTag,
		GitTreeState: gitTreeState,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func PrintVersion(version Version) {
	fmt.Printf("BuildInfo: %s\n", version.Version)
	fmt.Printf("  BuildDate: %s\n", version.BuildDate)
	fmt.Printf("  GitCommit: %s\n", version.GitCommit)
	fmt.Printf("  GitTreeState: %s\n", version.GitTreeState)
	if version.GitTag != "" {
		fmt.Printf("  GitTag: %s\n", version.GitTag)
	}
	fmt.Printf("  GoVersion: %s\n", version.GoVersion)
	fmt.Printf("  Compiler: %s\n", version.Compiler)
	fmt.Printf("  Platform: %s\n", version.Platform)
}
