// Copyright 2019 Yunion
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

package apis

type VirtualResourceDetails struct {
	ModelBaseDetails
}

type VirtualResourceListInput struct {
	StatusStandaloneResourceListInput

	ProjectizedResourceListInput

	// 列表中包含标记为"系统资源"的资源
	System *bool `json:"system"`
	// 是否显示回收站内的资源，默认不显示（对实现了回收站的资源有效，例如主机，磁盘，镜像）
	PendingDelete *bool `json:"pending_delete"`
	// 是否显示所有资源，包括回收站和不再回收站的资源
	// TODO: fix this???
	PendingDeleteAll *bool `json:"-"`
}
