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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
)

func (this *ProcessDeployment) PrepareRequest(token auth.Token, processId string) (deployment deploymentmodel.Deployment, err error) {
	req, err := http.NewRequest("GET", this.config.ProcessDeploymentUrl+"/v3/prepared-deployments/"+url.PathEscape(processId)+"?with_options=false", nil)
	if err != nil {
		return deployment, err
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return deployment, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		return deployment, err
	}
	err = json.NewDecoder(resp.Body).Decode(&deployment)
	return deployment, err
}

func (this *ProcessDeployment) Deploy(token auth.Token, deployment deploymentmodel.Deployment, allowMissingServiceSelection bool, hubId string) (result deploymentmodel.Deployment, err error) {
	queryStr := ""
	query := url.Values{}
	source := this.config.ProcessDeploymentSource
	if source != "" {
		query.Set("source", source)
	}
	if allowMissingServiceSelection {
		query.Set("optional_service_selection", "true")
	}
	if len(query) > 0 {
		queryStr = "?" + query.Encode()
	}
	for i, element := range deployment.Elements {
		if element.Task != nil && len(element.Task.Selection.SelectionOptions) > 0 {
			element.Task.Selection.SelectionOptions = []deploymentmodel.SelectionOption{}
		}
		if element.MessageEvent != nil && len(element.MessageEvent.Selection.SelectionOptions) > 0 {
			element.MessageEvent.Selection.SelectionOptions = []deploymentmodel.SelectionOption{}
		}
		if element.ConditionalEvent != nil && len(element.ConditionalEvent.Selection.SelectionOptions) > 0 {
			element.ConditionalEvent.Selection.SelectionOptions = []deploymentmodel.SelectionOption{}
		}
		deployment.Elements[i] = element
	}
	body := new(bytes.Buffer)
	err = json.NewEncoder(body).Encode(deployment)
	if err != nil {
		this.libConfig.GetLogger().Error("error in ProcessDeployment.Deploy", "error", err, "stack", string(debug.Stack()))
		return result, err
	}
	endpoint := this.config.ProcessDeploymentUrl + "/v3/deployments" + queryStr
	if hubId != "" {
		endpoint = this.config.FogProcessDeploymentUrl + "/deployments/" + url.PathEscape(hubId) + queryStr
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		return result, err
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

var DefaultTimeout = 30 * time.Second

func (this *ProcessDeployment) CheckDeployment(token auth.Token, deploymentId string) (int, error) {
	client := http.Client{
		Timeout: DefaultTimeout,
	}
	req, err := http.NewRequest(
		"GET",
		this.config.ProcessDeploymentUrl+"/v3/deployments/"+url.PathEscape(deploymentId),
		nil,
	)
	if err != nil {
		this.libConfig.GetLogger().Error("error in CheckDeployment", "error", err, "stack", string(debug.Stack()))
		return 0, err
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("X-UserId", token.GetUserId())

	this.libConfig.GetLogger().Debug("check deployment request", "url", req.URL.String(), "method", req.Method, "token", req.Header.Get("Authorization"), "xuser", req.Header.Get("X-UserId"))

	resp, err := client.Do(req)
	if err != nil {
		this.libConfig.GetLogger().Error("error in CheckDeployment", "error", err, "stack", string(debug.Stack()))
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func (this *ProcessDeployment) CheckFogDeployment(token auth.Token, hubId string, deploymentId string) (error, error) {
	metadata, err, _ := this.GetFogSyncMetadata(token, hubId, deploymentId)
	if err != nil {
		return nil, err
	}
	if len(metadata) == 0 {
		return fmt.Errorf("no matching fog process deployment found"), nil
	}
	if metadata[0].MarkedForDelete {
		return fmt.Errorf("fog deployment is marked for deletion"), nil
	}
	if metadata[0].IsPlaceholder {
		return fmt.Errorf("fog deployment is marked as placeholder"), nil
	}
	return nil, nil
}

func (this *ProcessDeployment) GetFogSyncMetadata(token auth.Token, hubId string, deploymentId string) (result []DeploymentMetadata, err error, code int) {
	req, err := http.NewRequest("GET", this.config.FogProcessSyncUrl+"/metadata/"+url.PathEscape(hubId)+"?deployment_id="+url.QueryEscape(deploymentId), nil)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	req.Header.Set("Authorization", token.Jwt())
	client := &http.Client{
		Timeout: DefaultTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		return result, err, resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure empty body to enable connection reuse and prevent memory leaks
		return result, err, http.StatusInternalServerError
	}
	return result, err, http.StatusOK
}

type DeploymentMetadata struct {
	Metadata
	SyncInfo
}

type Metadata struct {
	CamundaDeploymentId string `json:"camunda_deployment_id"`
}

type SyncInfo struct {
	NetworkId       string    `json:"network_id"`
	IsPlaceholder   bool      `json:"is_placeholder"`
	MarkedForDelete bool      `json:"marked_for_delete"`
	SyncDate        time.Time `json:"sync_date"`
}
