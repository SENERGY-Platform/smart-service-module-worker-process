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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"time"
)

func New(ctx context.Context, wg *sync.WaitGroup, config Config, libConfig configuration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo) (*ProcessDeployment, error) {
	err := StartDoneEventHandling(ctx, wg, config, libConfig)
	if err != nil {
		return nil, err
	}
	return &ProcessDeployment{config: config, libConfig: libConfig, auth: auth, smartServiceRepo: smartServiceRepo}, nil
}

type ProcessDeployment struct {
	config           Config
	libConfig        configuration.Config
	auth             *auth.Auth
	smartServiceRepo SmartServiceRepo
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
	UseModuleDeleteInfo(info model.ModuleDeleteInfo) error
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

	isFogDeployment, hubId, err := this.IsFogDeployment(token, task, deployment)
	if err != nil {
		log.Println("ERROR: unable to use variables", err)
		return modules, outputs, err
	}

	resultDeployment, err := this.Deploy(token, deployment, true, hubId)
	if err != nil {
		log.Println("ERROR: unable to deploy process", err)
		return modules, outputs, err
	}

	moduleData := this.getModuleData(task)
	moduleData["process_deployment_name"] = resultDeployment.Name
	moduleData["process_deployment_id"] = resultDeployment.Id
	if isFogDeployment {
		moduleData["is_fog_deployment"] = true
		moduleData["fog_hub"] = hubId
	}

	deleteEndpoint := this.config.ProcessDeploymentUrl + "/v3/deployments/" + url.PathEscape(resultDeployment.Id)
	if isFogDeployment {
		deleteEndpoint = this.config.FogProcessDeploymentUrl + "/deployments/" + url.PathEscape(hubId) + "/" + url.PathEscape(resultDeployment.Id)
	}

	outputs = map[string]interface{}{
		"process_deployment_id": resultDeployment.Id,
		"done_event":            deploymentIdToEventId(resultDeployment.Id),
	}
	if isFogDeployment {
		outputs["is_fog_deployment"] = true
		outputs["fog_hub"] = hubId
	}

	return []model.Module{{
			Id:               this.getModuleId(task),
			ProcesInstanceId: task.ProcessInstanceId,
			SmartServiceModuleInit: model.SmartServiceModuleInit{
				DeleteInfo: &model.ModuleDeleteInfo{
					Url:    deleteEndpoint,
					UserId: userId,
				},
				ModuleType: this.libConfig.CamundaWorkerTopic,
				ModuleData: moduleData,
			},
		}},
		outputs,
		err
}

func (this *ProcessDeployment) Undo(modules []model.Module, reason error) {
	log.Println("UNDO:", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil {
			err := this.smartServiceRepo.UseModuleDeleteInfo(*module.DeleteInfo)
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
		this.setMsgEventConfig,
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

func (this *ProcessDeployment) IsFogDeployment(token auth.Token, task model.CamundaExternalTask, deployment deploymentmodel.Deployment) (isFogDeployment bool, hubId string, err error) {
	preferFogDeployment, err := this.getPreferFogDeployment(task)
	if err != nil {
		return false, "", err
	}
	if !preferFogDeployment {
		return false, "", nil
	}

	devices := []string{}
	groups := []string{}
	imports := []string{}
	usesEvents := false
	for _, element := range deployment.Elements {
		if element.Task != nil {
			if element.Task.Selection.SelectedDeviceId != nil {
				devices = append(devices, *element.Task.Selection.SelectedDeviceId)
			}
			if element.Task.Selection.SelectedDeviceGroupId != nil {
				groups = append(groups, *element.Task.Selection.SelectedDeviceGroupId)
			}
			if element.Task.Selection.SelectedImportId != nil {
				imports = append(imports, *element.Task.Selection.SelectedImportId)
			}
		}
		if element.MessageEvent != nil {
			usesEvents = true
			if element.MessageEvent.Selection.SelectedDeviceId != nil {
				devices = append(devices, *element.MessageEvent.Selection.SelectedDeviceId)
			}
			if element.MessageEvent.Selection.SelectedDeviceGroupId != nil {
				groups = append(groups, *element.MessageEvent.Selection.SelectedDeviceGroupId)
			}
			if element.MessageEvent.Selection.SelectedImportId != nil {
				imports = append(imports, *element.MessageEvent.Selection.SelectedImportId)
			}
		}
	}
	if !this.config.AllowEventsInFogProcesses && usesEvents {
		return false, "", nil
	}
	if !this.config.AllowImportsInFogProcesses && len(imports) > 0 {
		return false, "", nil
	}
	for _, groupId := range groups {
		group, err := this.GetGroup(token, groupId)
		if err != nil {
			return false, "", err
		}
		devices = append(devices, group.DeviceIds...)
	}
	networks, err := this.GetFogNetworks(token)
	if err != nil {
		return false, "", err
	}
	if len(devices) == 0 {
		log.Println("WARNING: process deployments without devices wont be run in fog")
		return false, "", nil
	}
	for _, network := range networks {
		networkIndex := map[string]bool{}
		for _, id := range network.DeviceIds {
			networkIndex[id] = true
		}
		missingDeviceInNetwork := false
		for _, id := range devices {
			if !networkIndex[id] {
				missingDeviceInNetwork = true
				break
			}
		}
		if !missingDeviceInNetwork {
			return true, network.Id, nil
		}
	}
	return false, "", nil
}

type Hub struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	DeviceLocalIds []string `json:"device_local_ids"`
	DeviceIds      []string `json:"device_ids"`
}

func (this *ProcessDeployment) GetFogNetworks(token auth.Token) (result []Hub, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest(
		"GET",
		this.config.FogProcessSyncUrl+"/networks",
		nil,
	)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	req.Header.Set("Authorization", token.Jwt())

	resp, err := client.Do(req)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		debug.PrintStack()
		temp, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	_, _ = io.ReadAll(resp.Body)
	return result, err
}

type DeviceGroup struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	DeviceIds []string `json:"device_ids"`
}

func (this *ProcessDeployment) GetGroup(token auth.Token, id string) (result DeviceGroup, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest(
		"GET",
		this.config.DeviceRepositoryUrl+"/device-groups/"+url.PathEscape(id),
		nil,
	)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	req.Header.Set("Authorization", token.Jwt())

	resp, err := client.Do(req)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		debug.PrintStack()
		temp, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	_, _ = io.ReadAll(resp.Body)
	return result, err
}
