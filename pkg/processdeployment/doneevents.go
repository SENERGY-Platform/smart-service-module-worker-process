/*
 * Copyright (c) 2023 InfAI (CC SES)
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
	"time"

	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
)

func triggerDoneEvent(libConfig configuration.Config, resourceId string) {
	go func() {
		time.Sleep(5 * time.Second)
		eventId := deploymentIdToEventId(resourceId)
		err := camunda.SendEventTrigger(libConfig, eventId, nil)
		if err != nil {
			libConfig.GetLogger().Error("unable to send event trigger", "error", err)
		}
	}()
}

func deploymentIdToEventId(id string) string {
	return "deployment_done_" + id
}
