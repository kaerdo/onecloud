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

package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SNatDEntryCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatDEntryCreateTask{})
}

func (self *SNatDEntryCreateTask) TaskFailed(ctx context.Context, dnatEntry models.INatHelper, err error) {
	dnatEntry.SetStatus(self.UserCred, api.NAT_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(dnatEntry, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	natgateway, err := dnatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_CREATE_DNAT, nil, self.UserCred, false)
	} else {
		logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_ALLOCATE, nil, self.UserCred, false)
	}
	self.SetStageFailed(ctx, err.Error())
}

func (self *SNatDEntryCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dnatEntry := obj.(*models.SNatDEntry)
	NatToBindIPStage(ctx, self, dnatEntry)
}

func (self *SNatDEntryCreateTask) OnBindIPCompleteFailed(ctx context.Context, dnatEntry *models.SNatDEntry,
	reason jsonutils.JSONObject) {

	self.TaskFailed(ctx, dnatEntry, fmt.Errorf(reason.String()))
}

func (self *SNatDEntryCreateTask) OnBindIPComplete(ctx context.Context, dnatEntry *models.SNatDEntry,
	body jsonutils.JSONObject) {

	cloudNatGateway, err := dnatEntry.GetINatGateway()
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, errors.Wrap(err, "Get NatGateway failed"))
		return
	}

	externalIPID, err := self.Params.GetString("eip_external_id")
	// construct a DNat RUle
	dnatRule := cloudprovider.SNatDRule{
		Protocol:     dnatEntry.IpProtocol,
		InternalIP:   dnatEntry.InternalIP,
		InternalPort: dnatEntry.InternalPort,
		ExternalIP:   dnatEntry.ExternalIP,
		ExternalIPID: externalIPID,
		ExternalPort: dnatEntry.ExternalPort,
	}
	extDnat, err := cloudNatGateway.CreateINatDEntry(dnatRule)
	if err != nil {
		if self.Params.Contains("need_bind") {
			err1 := CreateINatFailedRollback(ctx, self, dnatEntry)
			if err1 != nil {
				eip_id, _ := self.Params.GetString("eip_id")
				log.Errorf("roll back after failing to create dnat in cloud so that eip %s need to sync with cloud", eip_id)
			}
		}
		self.TaskFailed(ctx, dnatEntry, errors.Wrapf(err, "Create DNat Entry '%s' failed", dnatEntry.ExternalId))
		return
	}

	err = cloudprovider.WaitStatus(extDnat, api.NAT_STAUTS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, err)
		return
	}

	err = db.SetExternalId(dnatEntry, self.UserCred, extDnat.GetGlobalId())
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, errors.Wrap(err, "set external id failed"))
		return
	}

	dnatEntry.SetStatus(self.UserCred, api.NAT_STAUTS_AVAILABLE, "")
	db.OpsLog.LogEvent(dnatEntry, db.ACT_ALLOCATE, dnatEntry.GetShortDesc(ctx), self.UserCred)
	natgateway, err := dnatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_CREATE_DNAT, nil, self.UserCred, true)
	} else {
		logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}
