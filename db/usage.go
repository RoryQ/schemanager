package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strings"
)

const (
	readmePath  = "db/README.md"
	migratePath = "db/migrate.go"
)

func main() {
	helpOutput, err := exec.Command("go", "run", migratePath, "--help").Output()
	if err != nil {
		panic(err)
	}

	readme, err := ioutil.ReadFile(readmePath)
	if err != nil {
		panic(err)
	}

	format := "<!--usage-shell-->\n```\n%s```"
	re := regexp.MustCompile(fmt.Sprintf(format, "[^`]+"))
	matches := re.FindStringSubmatch(string(readme))
	replaced := strings.ReplaceAll(string(readme), matches[0],
		fmt.Sprintf(format, helpOutput))

	err = ioutil.WriteFile(readmePath, []byte(replaced), 0644)
	if err != nil {
		panic(err)
	}
}
