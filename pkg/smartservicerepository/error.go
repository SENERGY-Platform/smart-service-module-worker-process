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

package smartservicerepository

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/model"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

func (this *SmartServiceRepository) SendWorkerError(task model.CamundaExternalTask, errMsg error) error {
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(errMsg.Error())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	req, err := http.NewRequest("PUT", this.config.SmartServiceRepositoryUrl+"/instances-by-process-id/"+url.PathEscape(task.ProcessInstanceId)+"/error", body)
	if err != nil {
		return err
	}
	token, err := this.auth.Ensure()
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}
