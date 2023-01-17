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

type Config struct {
	ProcessDeploymentUrl       string `json:"process_deployment_url"`
	FogProcessDeploymentUrl    string `json:"fog_process_deployment_url"`
	FogProcessSyncUrl          string `json:"fog_process_sync_url"`
	DeviceRepositoryUrl        string `json:"device_repository_url"`
	AllowEventsInFogProcesses  bool   `json:"allow_events_in_fog_processes"`
	AllowImportsInFogProcesses bool   `json:"allow_imports_in_fog_processes"`
	ProcessDeploymentSource    string `json:"process_deployment_source"`
	WorkerParamPrefix          string `json:"worker_param_prefix"`
	KafkaUrl                   string `json:"kafka_url"`
	KafkaConsumerGroup         string `json:"kafka_consumer_group"`
	ProcessDeploymentDoneTopic string `json:"process_deployment_done_topic"`
}
