// Copyright © 2021 DataHen Canada Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/DataHenHQ/till/cmd"
)

var (
	version = ""
	commit  = ""
	date    = ""
	baseurl = ""
	pubkey  = ""
)

func main() {
	if version != "" {
		cmd.ReleaseVersion = version
	}
	if commit != "" {
		cmd.ReleaseCommit = commit
	}
	if date != "" {
		cmd.ReleaseDate = date
	}
	if baseurl != "" {
		cmd.BaseURL = baseurl
	}
	if pubkey != "" {
		cmd.PubKey = pubkey
	}
	cmd.Execute()
}
