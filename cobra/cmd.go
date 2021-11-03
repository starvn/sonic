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

// Package cobra defines the cobra command structs and an execution method for adding an improved CLI to Sonic based API Gateways
package cobra

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starvn/turbo/config"
	"os"
	"sync"
	"time"
)

type Executor func(config.ServiceConfig)

func Execute(configParser config.Parser, f Executor) {
	ExecuteRoot(configParser, f, DefaultRoot)
}

func ExecuteRoot(configParser config.Parser, f Executor, root Root) {
	root.Build()
	root.Execute(configParser, f)
}

func GetConfigFlag() string {
	return cfgFile
}

func GetDebugFlag() bool {
	return debug > 0
}

type FlagBuilder func(*cobra.Command)

func StringFlagBuilder(dst *string, long, short, defaultValue, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().StringVarP(dst, long, short, defaultValue, help)
	}
}

func BoolFlagBuilder(dst *bool, long, short string, defaultValue bool, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().BoolVarP(dst, long, short, defaultValue, help)
	}
}

func DurationFlagBuilder(dst *time.Duration, long, short string, defaultValue time.Duration, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().DurationVarP(dst, long, short, defaultValue, help)
	}
}

func Float64FlagBuilder(dst *float64, long, short string, defaultValue float64, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().Float64VarP(dst, long, short, defaultValue, help)
	}
}

func IntFlagBuilder(dst *int, long, short string, defaultValue int, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().IntVarP(dst, long, short, defaultValue, help)
	}
}

func CountFlagBuilder(dst *int, long, short string, help string) FlagBuilder {
	return func(cmd *cobra.Command) {
		cmd.PersistentFlags().CountVarP(dst, long, short, help)
	}
}

type Command struct {
	Cmd   *cobra.Command
	Flags []FlagBuilder
	once  *sync.Once
}

func NewCommand(command *cobra.Command, flags ...FlagBuilder) Command {
	return Command{Cmd: command, Flags: flags, once: new(sync.Once)}
}

func (c *Command) AddFlag(f FlagBuilder) {
	c.Flags = append(c.Flags, f)
}

func (c *Command) BuildFlags() {
	c.once.Do(func() {
		for i := range c.Flags {
			c.Flags[i](c.Cmd)
		}
	})
}

func (c *Command) AddSubCommand(cmd *cobra.Command) {
	c.Cmd.AddCommand(cmd)
}

func NewRoot(root Command, subCommands ...Command) Root {
	r := Root{Command: root, SubCommands: subCommands, once: new(sync.Once)}
	return r
}

type Root struct {
	Command
	SubCommands []Command
	once        *sync.Once
}

func (r *Root) Build() {
	r.once.Do(func() {
		r.BuildFlags()
		for i := range r.SubCommands {
			r.SubCommands[i].BuildFlags()
			r.Cmd.AddCommand(r.SubCommands[i].Cmd)
		}
	})
}

func (r Root) Execute(configParser config.Parser, f Executor) {
	parser = configParser
	run = f
	if err := r.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
