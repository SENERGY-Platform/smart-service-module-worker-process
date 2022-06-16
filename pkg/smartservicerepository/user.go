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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
)

func (this *SmartServiceRepository) GetInstanceUser(instanceId string) (userId string, err error) {
	req, err := http.NewRequest("GET", this.config.SmartServiceRepositoryUrl+"/instances-by-process-id/"+url.PathEscape(instanceId)+"/user-id", nil)
	if err != nil {
		return userId, err
	}
	token, err := this.auth.Ensure()
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userId, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		return userId, err
	}
	err = json.NewDecoder(resp.Body).Decode(&userId)
	if err != nil {
		return userId, err
	}
	return userId, nil
}
