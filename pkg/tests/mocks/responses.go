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

package mocks

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"sync"
)

func MockResponses(ctx context.Context, wg *sync.WaitGroup, responses []Response) (url string) {
	router := httprouter.New()

	for _, resp := range responses {
		msg := resp.Message
		if resp.Method == "GET" {
			router.GET(resp.Endpoint, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
				writer.Write([]byte(msg))
			})
		}
		if resp.Method == "POST" {
			router.POST(resp.Endpoint, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
				writer.Write([]byte(msg))
			})
		}
		if resp.Method == "PUT" {
			router.POST(resp.Endpoint, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
				writer.Write([]byte(msg))
			})
		}
	}

	server := httptest.NewServer(router)
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}
