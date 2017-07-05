package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/heppu/cvm/git"
)

const (
	REFS            = "https://chromium.googlesource.com/chromium/src/+refs?format=JSON"
	POSITION_LOOKUP = "https://omahaproxy.appspot.com/deps.json?version=%s"

	BASE_URL      = "https://www.googleapis.com/storage/v1/b/chromium-browser-snapshots/"
	PLATFORMS_URL = BASE_URL + "o?delimiter=/&prefix=&fields=prefixes"
	BUILDS_URL    = BASE_URL + "o?delimiter=/&prefix=%s&fields=items(updated),prefixes,nextPageToken&pageToken=%s"
	FILES_URL     = BASE_URL + "o?delimiter=/&prefix=%s&fields=items(mediaLink,metadata,name,size,updated)"
)

type PlatformsResponse struct {
	Prefixes []string `json:"prefixes"`
}

type BuildsResponse struct {
	NextPageToken string   `json:"nextPageToken"`
	Prefixes      []string `json:"prefixes"`
	Items         []struct {
		Updated time.Time `json:"updated"`
	} `json:"items"`
}

type FilesResponse struct {
	Items []FileInfo `json:"items"`
}

type FileInfo struct {
	Name      string    `json:"name"`
	Updated   time.Time `json:"updated"`
	Size      string    `json:"size"`
	MediaLink string    `json:"mediaLink"`
	Metadata  Metadata  `json:"metadata"`
}

type Metadata struct {
	CrCommitPositionNumber string `json:"cr-commit-position-number"`
	CrGitCommit            string `json:"cr-git-commit"`
	CrCommitPosition       string `json:"cr-commit-position"`
}

type Revisions struct {
	ChromiumRevision string `json:"chromium_revision"`
	WebkitRevision   string `json:"webkit_revision"`
	V8Revision       string `json:"v8_revision"`
	V8RevisionGit    string `json:"v8_revision_git"`
}

type VersionInfo struct {
	ChromiumVersion      string `json:"chromium_version"`
	SkiaCommit           string `json:"skia_commit"`
	ChromiumBasePosition string `json:"chromium_base_position"`
	V8Version            string `json:"v8_version"`
	ChromiumBranch       string `json:"chromium_branch"`
	V8Position           string `json:"v8_position"`
	ChromiumBaseCommit   string `json:"chromium_base_commit"`
	ChromiumCommit       string `json:"chromium_commit"`
}

type VersionsResponse []struct {
	Os       string     `json:"os"`
	Versions []Versions `json:"versions"`
}

type Versions struct {
	BranchCommit       string `json:"branch_commit"`
	BranchBasePosition string `json:"branch_base_position"`
	SkiaCommit         string `json:"skia_commit"`
	V8Version          string `json:"v8_version"`
	PreviousVersion    string `json:"previous_version"`
	V8Commit           string `json:"v8_commit"`
	TrueBranch         string `json:"true_branch"`
	PreviousReldate    string `json:"previous_reldate"`
	BranchBaseCommit   string `json:"branch_base_commit"`
	Version            string `json:"version"`
	CurrentReldate     string `json:"current_reldate"`
	CurrentVersion     string `json:"current_version"`
	Os                 string `json:"os"`
	Channel            string `json:"channel"`
	ChromiumCommit     string `json:"chromium_commit"`
}

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}

func (c *Client) GetVersionInfo(cv git.ChromeVersion) (info VersionInfo, err error) {
	url := fmt.Sprintf(POSITION_LOOKUP, cv)
	fmt.Println(url)
	err = c.getJson(url, &info)
	return
}

func (c *Client) GetAll() error {
	platforms, err := c.GetPlatforms()
	if err != nil {
		return err
	}
	wg := &sync.WaitGroup{}
	for _, platform := range platforms {
		wg.Add(1)
		go func(platform string) {
			log.Printf("Indexing builds for %s...\n", platform)
			builds, lastUpdated, err := c.GetAllBuildsForPlatform(platform)
			if err != nil {
				log.Printf("%s failed: %s\n", err)
			} else {
				log.Printf("%s ready, %d builds indexed, last updated %s\n", platform, len(builds), lastUpdated)
			}
			wg.Done()
		}(platform)

	}
	wg.Wait()
	return nil
}

func (c *Client) GetPlatforms() (platforms []string, err error) {
	data := &PlatformsResponse{}
	err = c.getJson(PLATFORMS_URL, data)
	for _, platform := range data.Prefixes {
		if platform == "tmp/" || platform == "gs-test/" || platform == "icons/" {
			continue
		}
		platforms = append(platforms, platform)
	}
	return
}

func (c *Client) GetAllBuildsForPlatform(platform string) (builds []string, lastUpdated time.Time, err error) {
	pageToken := ""
	for {
		data := &BuildsResponse{}
		if err = c.getJson(fmt.Sprintf(BUILDS_URL, platform, pageToken), data); err != nil {
			return
		}
		builds = append(builds, data.Prefixes...)
		if data.NextPageToken != "" {
			pageToken = data.NextPageToken
		} else {
			if len(data.Items) == 1 {
				lastUpdated = data.Items[0].Updated
			}
			break
		}
	}
	return
}

func (c *Client) GetFilesForBuild(build string) (items []FileInfo, err error) {
	data := &FilesResponse{}
	err = c.getJson(fmt.Sprintf(FILES_URL, build), data)
	items = data.Items
	return
}

func (c *Client) GetBuildInfo(build string) (fr FilesResponse, err error) {
	url := fmt.Sprintf(FILES_URL, build)
	err = c.getJson(url, &fr)
	return
}

func (c *Client) GetRevisions(build string) (revisions Revisions, err error) {
	data := &FilesResponse{}
	if err = c.getJson(fmt.Sprintf(FILES_URL, build), data); err != nil {
		return
	}

	var revUrl string
	for _, fi := range data.Items {
		if strings.HasSuffix(fi.Name, "REVISIONS") {
			revUrl = fi.MediaLink
			break
		}
	}
	if revUrl == "" {
		err = fmt.Errorf("No revision url found")
		return
	}

	c.getJson(revUrl, &revisions)
	return
}

func (c *Client) GetZip(build string) (file string, err error) {
	data := &FilesResponse{}
	url := fmt.Sprintf(FILES_URL, build)
	fmt.Println(url)
	if err = c.getJson(url, data); err != nil {
		return
	}
	fmt.Println(data)
	var fileUrl string
	for _, fi := range data.Items {
		fmt.Println(fi.Name)
		if strings.HasSuffix(fi.Name, "linux.zip") {
			fileUrl = fi.MediaLink
			break
		}
	}
	if fileUrl == "" {
		err = fmt.Errorf("No linux.zip url found")
		return
	}

	file = fileUrl
	return
}

func (c *Client) getJson(url string, data interface{}) error {
	res, err := c.client.Get(url)
	if err != nil {
		return err
	}

	if err = json.NewDecoder(res.Body).Decode(data); err != nil {
		data, err := ioutil.ReadAll(res.Body)
		fmt.Println("adasd", err)
		fmt.Println(string(data))
	}
	res.Body.Close()
	return err
}

func (c *Client) get(url string) (data []byte, err error) {
	var res *http.Response
	if res, err = c.client.Get(url); err != nil {
		return
	}

	data, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	return
}
