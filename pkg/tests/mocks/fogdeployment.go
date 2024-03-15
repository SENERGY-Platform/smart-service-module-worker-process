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

package mocks

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/service-commons/pkg/accesslog"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

func NewFogDeploymentMock() *FogDeploymentMock {
	return &FogDeploymentMock{requestsLog: []Request{}}
}

type FogDeploymentMock struct {
	requestsLog         []Request
	preparedDeployments map[string]deploymentmodel.Deployment
	mux                 sync.Mutex
}

func (this *FogDeploymentMock) PopRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	this.requestsLog = []Request{}
	return result
}

func (this *FogDeploymentMock) logRequest(r Request) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.requestsLog = append(this.requestsLog, r)
}

func (this *FogDeploymentMock) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *FogDeploymentMock) getRouter() http.Handler {
	router := httprouter.New()

	router.POST("/deployments/:hubid", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		deployment := deploymentmodel.Deployment{}
		err := json.Unmarshal(temp, &deployment)
		if err != nil {
			log.Println("ERROR: unable to parse request", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		optionals := map[string]bool{}
		optionalServiceStr := params.ByName("optional_service_selection")
		if optionalServiceStr != "" {
			optionals["service"], err = strconv.ParseBool(optionalServiceStr)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		deployment.Id = "new-deployment-id"
		err = deployment.Validate(deploymentmodel.ValidateRequest, optionals)
		if err != nil {
			log.Println("ERROR: bad request", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(writer).Encode(deployment)
	})

	return accesslog.New(router)
}
