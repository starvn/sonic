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
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/starvn/sonic/cobra/dumper"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"os"
	"time"
)

func errorMsg(content string) string {
	return dumper.ColorRed + content + dumper.ColorReset
}

func checkFunc(cmd *cobra.Command, args []string) {
	if cfgFile == "" {
		cmd.Println(errorMsg("Please, provide the path to your config file"))
		return
	}

	cmd.Printf("Parsing configuration file: %s\n", cfgFile)
	v, err := parser.Parse(cfgFile)
	if err != nil {
		cmd.Println(errorMsg("ERROR parsing the configuration file:") + fmt.Sprintf("\t%s\n", err.Error()))
		os.Exit(1)
		return
	}

	if debug > 0 {
		cc := dumper.New(cmd, checkDumpPrefix, debug)
		if err := cc.Dump(v); err != nil {
			cmd.Println(errorMsg("ERROR checking the configuration file:") + fmt.Sprintf("\t%s\n", err.Error()))
			os.Exit(1)
			return
		}
	}

	if checkGinRoutes {
		if err := runRouter(v); err != nil {
			cmd.Println(errorMsg("ERROR testing the configuration file:") + fmt.Sprintf("\t%s\n", err.Error()))
			os.Exit(1)
			return
		}
	}

	cmd.Printf("%sSyntax OK!%s\n", dumper.ColorGreen, dumper.ColorReset)
}

func runRouter(cfg config.ServiceConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(r.(string))
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	cfg.Debug = cfg.Debug || debug > 0
	if port != 0 {
		cfg.Port = port
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	sgin.DefaultFactory(proxy.DefaultFactory(log.NoOp), log.NoOp).NewWithContext(ctx).Run(cfg)
	return nil
}
