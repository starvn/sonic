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

package main

import (
	"flag"
	"fmt"
	"github.com/starvn/sonic/test"
	"os"
)

func main() {
	flag.Parse()

	runner, tcs, err := test.NewIntegration(nil, nil, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	defer runner.Close()

	errors := 0

	for _, tc := range tcs {
		if err := runner.Check(tc); err != nil {
			errors++
			fmt.Printf("%s: %s\n", tc.Name, err.Error())
			continue
		}
		fmt.Printf("%s: ok\n", tc.Name)
	}
	fmt.Printf("%d test completed\n", len(tcs))

	if errors == 0 {
		return
	}

	fmt.Printf("%d test failed\n", errors)
	os.Exit(1)
}
