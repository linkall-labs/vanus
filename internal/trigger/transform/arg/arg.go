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

package arg

import (
	"strings"

	"github.com/linkall-labs/vanus/internal/trigger/context"
)

type Type uint8

const (
	Constant Type = iota
	EventAttribute
	EventData
	Define
)

func (t Type) String() string {
	switch t {
	case Constant:
		return "Constant"
	case EventAttribute:
		return "EventAttribute"
	case EventData:
		return "EventData"
	case Define:
		return "Define"
	}
	return "unknown"
}

type Arg interface {
	Type() Type
	Name() string
	// Evaluate arg value
	Evaluate(ceCtx *context.EventContext) (interface{}, error)
}

func NewArg(arg interface{}) (Arg, error) {
	switch arg.(type) {
	case string:
		argName := arg.(string)
		argName = strings.TrimSpace(argName)
		argLen := len(argName)
		if argLen >= 6 && argName[:6] == "$.data" {
			return newEventData(argName), nil
		}
		if argLen >= 2 && argName[:2] == "$." {
			return newEventAttribute(argName)
		}
		if argLen >= 3 && argName[0] == '<' && argName[argLen-1] == '>' {
			return newDefine(argName), nil
		}

	}
	return newConstant(arg), nil
}
