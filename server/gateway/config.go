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

package gateway

import (
	// first-party libraries.
	"github.com/vanus-labs/vanus/observability"

	// this project.
	"github.com/vanus-labs/vanus/internal/gateway"
	"github.com/vanus-labs/vanus/internal/primitive"
)

type Config struct {
	gateway.Config `yaml:",inline"`

	Observability observability.Config `yaml:"observability"`
}

func InitConfig(filename string) (*Config, error) {
	c := new(Config)
	err := primitive.LoadConfig(filename, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
