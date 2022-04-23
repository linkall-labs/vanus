// Copyright 2022 Linkall Inc.
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

package store

import (
	"github.com/linkall-labs/vanus/internal/primitive"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/internal/util"
)

type Config struct {
	ControllerAddresses []string   `yaml:"controllers"`
	IP                  string     `yaml:"ip"`
	Port                int        `yaml:"port"`
	Volume              VolumeInfo `yaml:"volume"`
}

type VolumeInfo struct {
	ID       vanus.ID `json:"id"`
	Dir      string   `json:"dir"`
	Capacity uint64   `json:"capacity"`
}

func InitConfig(filename string) (*Config, error) {
	c := new(Config)
	err := primitive.LoadConfig(filename, c)
	if err != nil {
		return nil, err
	}
	if c.IP == "" {
		c.IP = util.LocalIp
	}
	return c, nil
}
