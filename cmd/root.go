/*
Copyright © 2020 COMCREATE <t.imagawa@comcreate-info.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "unpm",
	Short: "Usage: unpm <command>",
	Long: `Usage: unpm <command>

where <command> is one of:
    config`,
	Run: func(cmd *cobra.Command, args []string) {
		callNpm(args)
		UpdateUpmConfig(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.unpmrc.yaml)")
	setCmd.PersistentFlags().StringP("userEmail", "e", getGitConfigUserEmail(), "Set the email to be used by Unity package manager")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".unpmrc" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".unpmrc")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func callNpm(args []string) {
	var cm *exec.Cmd

	if args == nil || len(args) == 0 {
		cm = exec.Command("npm")
	} else if len(args) == 1 {
		cm = exec.Command("npm", args[0])
	} else if len(args) == 2 {
		cm = exec.Command("npm", args[0], args[1])
	} else if len(args) == 3 {
		cm = exec.Command("npm", args[0], args[1], args[2])
	} else {
		fmt.Println("Not Support.")
		os.Exit(1)
		return
	}
	_ = cm.Run()
}

func UpdateUpmConfig(cmd *cobra.Command) {
	userEmail, _ := cmd.PersistentFlags().GetString("userEmail")
	config := getUserConfig()
	npmrc := getConfigFile(config, ".npmrc")
	unityConfigs := getConfigUsers(npmrc, userEmail)
	var lines []string
	for _, c := range unityConfigs {
		configText := createUnityConfigText(c)
		lines = append(lines, configText)
	}
	upmconfig := getWriteConfigFile(config, ".upmconfig.toml")
	saveLines(upmconfig, lines)
}

func getUserConfig() string {
	out, _ := exec.Command("npm", "config", "get", "userconfig").Output()
	return string(out)
}

func getConfigFile(userConfig string, fileName string) *os.File {
	f, err := os.Open(filepath.Join(filepath.Dir(userConfig), fileName))
	if err != nil {
		fmt.Println("error: " + err.Error())
		os.Exit(1)
		return nil
	}
	if f == nil {
		fmt.Println("error")
		os.Exit(1)
		return nil
	}
	return f
}

func getWriteConfigFile(userConfig string, fileName string) *os.File {
	f, err := os.OpenFile(filepath.Join(filepath.Dir(userConfig), fileName), os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModeAppend)
	if err != nil {
		fmt.Println("error: " + err.Error())
		os.Exit(1)
		return nil
	}
	if f == nil {
		fmt.Println("error")
		os.Exit(1)
		return nil
	}
	return f
}

func getConfigFileLines(file *os.File) []string {
	defer file.Close()
	if file == nil {
		fmt.Println("error")
		os.Exit(1)
		return nil
	}
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if n == 0 {
		return nil
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return nil
	}
	lines := string(buf[:n])
	return strings.Split(lines, "\n")
}

func getGitConfigUserEmail() string {
	out, _ := exec.Command("git", "config", "user.email").Output()
	lines := strings.Split(string(out), "\n")
	return lines[0]
}

func getConfigUsers(npmrc *os.File, userEmail string) map[string]unityConfig {
	unityConfigs := make(map[string]unityConfig)
	sliceLines := getConfigFileLines(npmrc)
	if sliceLines == nil {
		fmt.Println("error")
		os.Exit(1)
		return nil
	}
	for _, line := range sliceLines {
		if strings.Contains(line, "/:_authToken=") {
			authTokenLines := strings.Split(line, "/:_authToken=")
			currentUnityConfig := unityConfigs[authTokenLines[0]]
			currentUnityConfig.authUrl = "\"https:" + authTokenLines[0] + "\""
			currentUnityConfig.authToken = authTokenLines[1]
			currentUnityConfig.userEmail = "\"" + userEmail + "\""
			unityConfigs[authTokenLines[0]] = currentUnityConfig
		}
		if strings.Contains(line, "/:always-auth=") {
			alwaysAuthLines := strings.Split(line, "/:always-auth=")
			currentUnityConfig := unityConfigs[alwaysAuthLines[0]]
			currentUnityConfig.alwaysAuth = alwaysAuthLines[1]
			unityConfigs[alwaysAuthLines[0]] = currentUnityConfig
		}
	}
	return unityConfigs
}

func createUnityConfigText(config unityConfig) string {
	if config.authUrl == "" {
		fmt.Println("error")
		os.Exit(1)
		return ""
	}
	return "[npmAuth." + config.authUrl + "]\n" +
		"token = " + config.authToken + "\n" +
		"email = " + config.userEmail + "\n" +
		"alwaysAuth = " + config.alwaysAuth
}

func saveLines(file *os.File, lines []string) {
	defer file.Close()
	output := ""
	for i := range lines {
		if i > 0 {
			output = output + "\n"
		}
		output = output + lines[i]
	}
	_, err := file.WriteString(output)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
}
