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

package pkg

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	lib "github.com/SENERGY-Platform/smart-service-module-worker-lib"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/smartservicerepository"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/processdeployment"
)

func Start(ctx context.Context, wg *sync.WaitGroup, config processdeployment.Config, libConfig configuration.Config) error {
	handlerFactory := func(auth *auth.Auth, smartServiceRepo *smartservicerepository.SmartServiceRepository) (camunda.Handler, error) {
		interval, err := time.ParseDuration(config.HealthCheckInterval)
		if err != nil {
			return nil, err
		}

		handler, err := processdeployment.New(ctx, wg, config, libConfig, auth, smartServiceRepo)
		if err != nil {
			return nil, err
		}

		healthCheck := func(module model.SmartServiceModule) (health error, err error) {
			token, err := auth.ExchangeUserToken(module.UserId)
			if err != nil {
				return nil, err
			}
			isFogDeployment, fogHubId, deploymentId, err := getDeploymentId(module.ModuleData)
			if err != nil {
				return nil, err
			}
			if isFogDeployment {
				return handler.CheckFogDeployment(token, fogHubId, deploymentId)
			}
			code, err := handler.CheckDeployment(token, deploymentId)
			if err != nil {
				return nil, err
			}
			if code >= 300 {
				return fmt.Errorf("process deployment health check returned status-code %v", code), nil
			}
			return nil, nil
		}
		moduleQuery := model.ModulQuery{TypeFilter: &libConfig.CamundaWorkerTopic}
		smartServiceRepo.StartHealthCheck(ctx, interval, moduleQuery, healthCheck) //timer loop
		smartServiceRepo.RunHealthCheck(moduleQuery, healthCheck)                  //initial check
		return handler, nil
	}
	return lib.Start(ctx, wg, libConfig, handlerFactory)
}

func getDeploymentId(moduleData map[string]interface{}) (isFogDeployment bool, fogHubId string, deploymentId string, err error) {
	deploymentId, ok := moduleData["process_deployment_id"].(string)
	if !ok {
		return false, "", deploymentId, fmt.Errorf("missing process_deployment_id in module data")
	}
	isFogDeplIntterface, ok := moduleData["is_fog_deployment"]
	if !ok {
		return false, "", deploymentId, nil
	}
	isFogDeployment, ok = isFogDeplIntterface.(bool)
	if !ok {
		return false, "", deploymentId, fmt.Errorf("is_fog_deployment in module data is not bool but %s", reflect.TypeOf(isFogDeplIntterface).String())
	}
	if !isFogDeployment {
		return false, "", deploymentId, nil
	}
	fogHubId, ok = moduleData["fog_hub"].(string)
	if !ok {
		return false, "", deploymentId, fmt.Errorf("missing fog_hub in module data")
	}
	return isFogDeployment, fogHubId, deploymentId, nil
}
