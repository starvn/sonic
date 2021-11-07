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

package client

import (
	"context"
	"github.com/starvn/sonic/usage"
)

type Options struct {
	ClusterID string
	ServerID  string
	URL       string
	Version   string
}

func StartReporter(ctx context.Context, opt Options) error {
	reporter, err := usage.New(opt.URL, opt.ClusterID, opt.ServerID, opt.Version)
	if err != nil {
		return err
	}

	go reporter.Report(ctx)

	return nil
}
