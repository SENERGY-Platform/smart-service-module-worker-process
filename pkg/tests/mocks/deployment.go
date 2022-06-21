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
	"github.com/SENERGY-Platform/process-deployment/lib/api/util"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

func NewDeploymentMock() *DeploymentMock {
	return &DeploymentMock{}
}

type DeploymentMock struct {
	requestsLog         []Request
	preparedDeployments map[string]deploymentmodel.Deployment
	mux                 sync.Mutex
}

func (this *DeploymentMock) SetPreparedDeployments(value map[string]deploymentmodel.Deployment) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.preparedDeployments = value
}

func (this *DeploymentMock) getPreparedDeployment(id string) (result deploymentmodel.Deployment, ok bool) {
	this.mux.Lock()
	defer this.mux.Unlock()
	result, ok = this.preparedDeployments[id]
	return
}

func (this *DeploymentMock) PopRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	this.requestsLog = []Request{}
	return result
}

func (this *DeploymentMock) logRequest(r Request) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.requestsLog = append(this.requestsLog, r)
}

func (this *DeploymentMock) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *DeploymentMock) getRouter() http.Handler {
	router := httprouter.New()

	router.GET("/v3/prepared-deployments/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		writer.WriteHeader(200)
		result, ok := this.getPreparedDeployment(params.ByName("id"))
		if !ok {
			http.Error(writer, "unknown model id", http.StatusNotFound)
			return
		}
		json.NewEncoder(writer).Encode(result)
	})

	router.POST("/v3/deployments", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

	router.DELETE("/v3/deployments/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		temp, _ := io.ReadAll(request.Body)
		this.logRequest(Request{
			Method:   request.Method,
			Endpoint: request.URL.Path,
			Message:  string(temp),
		})
		writer.WriteHeader(200)
	})

	return util.NewLogger(router)
}
