package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"sync"

	"github.com/heppu/cvm/client"
	"github.com/heppu/cvm/git"
)

func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	baseDir := path.Join(usr.HomeDir, ".cvm")
	if err = os.Mkdir(baseDir, 0751); err != nil && os.IsNotExist(err) {
		panic(err)
	}

	binDir := path.Join(baseDir, "bin")
	if err = os.Mkdir(binDir, 0751); err != nil && os.IsNotExist(err) {
		panic(err)
	}

}

func main() {
	c := client.NewClient()

	asd, err := git.GetHashMap()
	for k, v := range asd {
		fmt.Println(k, v)
	}

	builds, lastUpdated, err := c.GetAllBuildsForPlatform("Linux_x64/")
	log.Println(err)
	log.Println(lastUpdated)

	wg := &sync.WaitGroup{}
	wg.Add(len(builds))
	ch := make(chan bool, 200)
	for _, build := range builds {
		ch <- true
		go func(build string) {
			defer func() {
				<-ch
				wg.Done()
			}()

			info, err := c.GetBuildInfo(build)
			if err != nil {
				fmt.Println(err)
				return
			}
			if info.Items[0].Metadata.CrGitCommit != "" {
				fmt.Printf("---\n%s\n%s\n---", build, info.Items[0].Metadata.CrGitCommit)
			}

		}(build)
	}
	wg.Wait()
	return

	cvs, err := git.GetVersions()
	log.Println(err)
	wg = &sync.WaitGroup{}
	wg.Add(len(cvs))
	ch = make(chan bool, 200)

	for _, cv := range cvs {
		ch <- true
		go func(cv git.ChromeVersion) {
			fmt.Println("---")

			if info, err := c.GetVersionInfo(cv); err != nil {
				log.Println(err)
			} else {
				if info.ChromiumBasePosition != "" {
					if fileUrl, err := c.GetZip("Linux_x64/" + info.ChromiumBasePosition + "/"); err != nil {
						log.Println(err)
					} else {
						fmt.Println(cv)
						fmt.Println(info.ChromiumBasePosition)
						fmt.Println(fileUrl)
					}
				}
			}
			<-ch
			wg.Done()
			fmt.Println("")
		}(cv)
	}
	wg.Wait()
	return

	if err := c.GetAll(); err != nil {
		log.Fatal(err)
	}
}
