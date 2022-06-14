/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package processdeployment

import (
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/model"
)

func New(config configuration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo) *ProcessDeployment {
	return &ProcessDeployment{config: config, auth: auth, smartServiceRepo: smartServiceRepo}
}

type ProcessDeployment struct {
	config           configuration.Config
	auth             *auth.Auth
	smartServiceRepo SmartServiceRepo
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
}

func (this *ProcessDeployment) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *ProcessDeployment) Undo(modules []model.Module, reason error) {
	//TODO implement me
	panic("implement me")
}
