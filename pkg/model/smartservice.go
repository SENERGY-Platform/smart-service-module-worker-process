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

package model

type Module struct {
	Id               string
	ProcesInstanceId string
	SmartServiceModuleInit
}

type SmartServiceModule struct {
	SmartServiceModuleBase `bson:",inline"`
	SmartServiceModuleInit `bson:",inline"`
}

type SmartServiceModuleBase struct {
	Id         string `json:"id"`
	UserId     string `json:"user_id" bson:"user_id"`
	InstanceId string `json:"instance_id" bson:"instance_id"`
	DesignId   string `json:"design_id" bson:"design_id"`
	ReleaseId  string `json:"release_id" bson:"release_id"`
}

type SmartServiceModuleInit struct {
	DeleteInfo *ModuleDeleteInfo      `json:"delete_info" bson:"delete_info"`
	ModuleType string                 `json:"module_type" bson:"module_type"` //"process-deployment" | "analytics" ...
	ModuleData map[string]interface{} `json:"module_data" bson:"module_data"`
}

type ModuleDeleteInfo struct {
	Url    string `json:"url" bson:"url"` //url receives a DELETE request and responds with a status code < 300 || code == 404 if ok
	UserId string `json:"user_id" bson:"user_id"`
}
