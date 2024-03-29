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
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
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
		log.Println("ERROR:", err)
		debug.PrintStack()
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
