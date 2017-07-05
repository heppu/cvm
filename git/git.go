package git

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type ChromeVersion struct {
	Major int
	Minor int
	Build int
	Patch int
}

func (c ChromeVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", c.Major, c.Minor, c.Build, c.Patch)
}

func GetVersions() (versions []ChromeVersion, err error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "http://chromium.googlesource.com/chromium/src")
	var stdout io.Reader
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, "refs/tags/"); i != -1 {
			if cv, err := ParseChromeVersionString(line[i+10:]); err == nil {
				versions = append(versions, *cv)
			}
		} else {
			fmt.Println(string(line))
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}
	return
}

func GetHashMap() (versions map[string]ChromeVersion, err error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "http://chromium.googlesource.com/chromium/src")
	var stdout io.Reader
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}

	versions = make(map[string]ChromeVersion)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, "refs/tags/"); i != -1 {
			if cv, err := ParseChromeVersionString(line[i+10:]); err == nil {
				versions[line[0:40]] = *cv
			}
		} else {
			fmt.Println(string(line))
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}
	return
}

func ParseChromeVersionString(str string) (*ChromeVersion, error) {
	err := fmt.Errorf("Could not parse chrome version from: %s", str)
	params := strings.Split(str, ".")
	if len(params) != 4 {
		return nil, err
	}
	nums := make([]int, 0, 4)
	for _, p := range params {
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		nums = append(nums, i)
	}
	return &ChromeVersion{nums[0], nums[1], nums[2], nums[3]}, nil
}
