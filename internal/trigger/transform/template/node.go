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

package template

type NodeType int

func (t NodeType) Type() NodeType {
	return t
}

const (
	Constant       NodeType = iota // Plain text.
	Variable                       // A <var> variable, example "key":<var>
	StringVariable                 // A <var> variable, example "key":"<var>" or "key":"other <var>"
)

type Node interface {
	Type() NodeType
	Value() string
}

type ConstantNode struct {
	NodeType
	Text string
}

func (p *parser) newConstant(text string) *ConstantNode {
	return &ConstantNode{Text: text, NodeType: Constant}
}

func (t *ConstantNode) Value() string {
	return t.Text
}

type VariableNode struct {
	NodeType
	Name string
}

func (p *parser) newVariable(name string, nodeType NodeType) *VariableNode {
	return &VariableNode{Name: name, NodeType: nodeType}
}

func (t *VariableNode) Value() string {
	return t.Name
}
