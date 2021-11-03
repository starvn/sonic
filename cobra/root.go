/*
 * Copyright (c) 2021 Huy Duc Dao
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cobra

import (
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/core"
)

var (
	cfgFile         string
	debug           int
	port            int
	checkGinRoutes  bool
	parser          config.Parser
	run             func(config.ServiceConfig)
	checkDumpPrefix = "\t"
	DefaultRoot     Root
	RootCommand     Command
	RunCommand      Command
	CheckCommand    Command

	rootCmd = &cobra.Command{
		Use:   "sonic",
		Short: "Sonic API Gateway builder",
	}

	checkCmd = &cobra.Command{
		Use:     "check",
		Short:   "Validates that the configuration file is valid.",
		Long:    "Validates that the active configuration file has a valid syntax to run the service.\nChange the configuration file by using the --config flag",
		Run:     checkFunc,
		Aliases: []string{"validate"},
		Example: "sonic check -d -c config.json",
	}

	runCmd = &cobra.Command{
		Use:     "run",
		Short:   "Run the Sonic server.",
		Long:    "Run the Sonic server.",
		Run:     runFunc,
		Example: "sonic run -d -c sonic.json",
	}
)

func init() {
	logo, err := base64.StdEncoding.DecodeString(encodedLogo)
	if err != nil {
		fmt.Println("decode error:", err)
	}
	cfgFlag := StringFlagBuilder(&cfgFile, "config", "c", "", "Path to the configuration filename")
	debugFlag := CountFlagBuilder(&debug, "debug", "d", "Enable the debug")
	RootCommand = NewCommand(rootCmd, cfgFlag, debugFlag)
	RootCommand.Cmd.SetHelpTemplate(string(logo) + "Version: " + core.SonicVersion + "\n\n" + rootCmd.HelpTemplate())

	ginRoutesFlag := BoolFlagBuilder(&checkGinRoutes, "test-gin-routes", "t", false, "Test the endpoint patterns against a real gin router on selected port")
	prefixFlag := StringFlagBuilder(&checkDumpPrefix, "indent", "i", checkDumpPrefix, "Indentation of the check dump")
	CheckCommand = NewCommand(checkCmd, ginRoutesFlag, prefixFlag)

	portFlag := IntFlagBuilder(&port, "port", "p", 0, "Listening port for the http service")
	RunCommand = NewCommand(runCmd, portFlag)

	DefaultRoot = NewRoot(RootCommand, CheckCommand, RunCommand)
}

const encodedLogo = "CiAgIF9fX19fX19fICBfICBfX19fX19fX19fXyAgX19fICAgX19fICBfX19fICBfX19fX19fXyBfX19fX19fX19fXyAgICAgIF9fX19fX18gIF9fCiAgLyBfXy8gX18gXC8gfC8gLyAgXy8gX19fLyAvIF8gfCAvIF8gXC8gIF8vIC8gX19fLyBfIC9fICBfXy8gX18vIHwgL3wgLyAvIF8gXCBcLyAvCiBfXCBcLyAvXy8gLyAgICAvLyAvLyAvX18gIC8gX18gfC8gX19fLy8gLyAgLyAoXyAvIF9fIHwvIC8gLyBfLyB8IHwvIHwvIC8gX18gfFwgIC8gCi9fX18vXF9fX18vXy98Xy9fX18vXF9fXy8gL18vIHxfL18vICAvX19fLyAgXF9fXy9fLyB8Xy9fLyAvX19fLyB8X18vfF9fL18vIHxffC9fLyAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCg=="
