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
	"errors"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/model"
	"log"
	"net/url"
	"runtime/debug"
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
	modelId := this.getProcessModelId(task)
	if modelId == "" {
		return modules, outputs, errors.New("missing process model id")
	}
	userId, err := this.smartServiceRepo.GetInstanceUser(task.ProcessInstanceId)
	if err != nil {
		log.Println("ERROR: unable to get instance user", err)
		return modules, outputs, err
	}
	token, err := this.auth.ExchangeUserToken(userId)
	if err != nil {
		log.Println("ERROR: unable to exchange user token", err)
		return modules, outputs, err
	}
	deployment, err := this.PrepareRequest(token, modelId)
	if err != nil {
		log.Println("ERROR: unable to prepare process deployment", err)
		return modules, outputs, err
	}
	err = this.UseVariables(task, &deployment)
	if err != nil {
		log.Println("ERROR: unable to use variables", err)
		return modules, outputs, err
	}
	resultDeployment, err := this.Deploy(token, deployment, true)
	if err != nil {
		log.Println("ERROR: unable to deploy process", err)
		return modules, outputs, err
	}

	moduleData := this.getModuleData(task)
	moduleData["process_deployment_name"] = resultDeployment.Name
	moduleData["process_deployment_id"] = resultDeployment.Id

	return []model.Module{{
			Id:               this.getModuleId(task),
			ProcesInstanceId: task.ProcessInstanceId,
			SmartServiceModuleInit: model.SmartServiceModuleInit{
				DeleteInfo: &model.ModuleDeleteInfo{
					Url:    this.config.ProcessDeploymentUrl + "/v3/deployments/" + url.PathEscape(resultDeployment.Id),
					UserId: userId,
				},
				ModuleType: this.config.CamundaWorkerTopic,
				ModuleData: moduleData,
			},
		}},
		map[string]interface{}{"process_deployment_id": resultDeployment.Id},
		err
}

func (this *ProcessDeployment) Undo(modules []model.Module, reason error) {
	log.Println("UNDO:", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil {
			err := this.useModuleDeleteInfo(*module.DeleteInfo)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
			}
		}
	}
}

func (this *ProcessDeployment) UseVariables(task model.CamundaExternalTask, deployment *deploymentmodel.Deployment) error {
	name := this.getProcessName(task)
	if name != "" {
		deployment.Name = name
	}
	handler := []func(task model.CamundaExternalTask, element *deploymentmodel.Element) error{
		this.setSelection,
		this.setParameter,
		this.setTime,
	}
	for i, element := range deployment.Elements {
		for _, h := range handler {
			err := h(task, &element)
			if err != nil {
				return err
			}
		}
		deployment.Elements[i] = element
	}
	return nil
}

func (this *ProcessDeployment) getModuleId(task model.CamundaExternalTask) string {
	return task.ProcessInstanceId + "." + task.Id
}
