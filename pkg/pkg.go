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
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/processdeployment"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/smartservicerepository"
	"sync"
)

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config) error {
	auth := auth.New(config)
	smartServiceRepo := smartservicerepository.New(config, auth)
	handler := processdeployment.New(config, auth, smartServiceRepo)
	camunda.Start(ctx, wg, config, smartServiceRepo, handler)
	return nil
}
