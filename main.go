package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"syscall"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/git"
	"golang.org/x/crypto/ssh/terminal"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func main() {
	cmd.Version = version

	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	viper.SetConfigName("lab.hcl")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(path.Join(home, ".config"))
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		legacyLoadConfig()
		writeConfig()
	}

	cmd.Execute()
}

const defaultGitLabHost = "https://gitlab.com"

func writeConfig() {
}

// legacyLoadConfig handles all of the credential setup and prompts for user
// input when not present
func legacyLoadConfig() (host, user, token string) {
	reader := bufio.NewReader(os.Stdin)
	var err error
	host, err = gitconfig.Entire("gitlab.host")
	if err != nil {
		fmt.Printf("Enter default GitLab host (default: %s): ", defaultGitLabHost)
		host, err = reader.ReadString('\n')
		host = strings.TrimSpace(host)
		if err != nil {
			log.Fatal(err)
		}
		if host == "" {
			host = defaultGitLabHost
		}
		cmd := git.New("config", "--global", "gitlab.host", host)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

	}
	var errt error
	user, err = gitconfig.Entire("gitlab.user")
	token, errt = gitconfig.Entire("gitlab.token")
	if err != nil {
		fmt.Print("Enter default GitLab user: ")
		user, err = reader.ReadString('\n')
		user = strings.TrimSpace(user)
		if err != nil {
			log.Fatal(err)
		}
		if user == "" {
			log.Fatal("git config gitlab.user must be set")
		}
		cmd := git.New("config", "--global", "gitlab.user", user)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		var tokenURL string
		if strings.HasSuffix(host, "/") {
			tokenURL = host + "profile/personal_access_tokens"
		} else {
			tokenURL = host + "/profile/personal_access_tokens"
		}

		// If the default user is being set this is the first time lab
		// is being run.
		if errt != nil {
			fmt.Printf("Create a token here: %s\nEnter default GitLab token (scope: api): ", tokenURL)
			byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatal(err)
			}
			token := strings.TrimSpace(string(byteToken))

			// Its okay for the key to be empty, since you can still call public repos
			if token != "" {
				cmd := git.New("config", "--global", "gitlab.token", token)
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	return
}
