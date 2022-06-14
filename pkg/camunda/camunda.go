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

package camunda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/model"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"time"
)

func New(config configuration.Config, smartServiceRepo SmartServiceRepository, handler Handler) *Camunda {
	return &Camunda{
		config:           config,
		handler:          handler,
		smartServiceRepo: smartServiceRepo,
	}
}

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, smartServiceRepo SmartServiceRepository, handler Handler) {
	New(config, smartServiceRepo, handler).Start(ctx, wg)
}

type Camunda struct {
	config           configuration.Config
	handler          Handler
	smartServiceRepo SmartServiceRepository
}

type SmartServiceRepository interface {
	SendWorkerError(task model.CamundaExternalTask, err error) error
	SendWorkerModules(modules []model.Module) (result []model.SmartServiceModule, err error)
}

type Handler interface {
	Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error)
	Undo(modules []model.Module, reason error)
}

func (this *Camunda) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			default:
				wait := this.executeNextTasks()
				if wait {
					duration := time.Duration(this.config.CamundaWorkerWaitDurationInMs) * time.Millisecond
					time.Sleep(duration)
				}
			}
		}
	}()
}

func (this *Camunda) executeNextTasks() (wait bool) {
	tasks, err := this.getTasks()
	if err != nil {
		log.Println("error on ExecuteNextTasks getTask", err)
		return true
	}
	if len(tasks) == 0 {
		return true
	}
	for _, task := range tasks {
		modules, outputs, err := this.handler.Do(task)
		if err != nil {
			repoErr := this.smartServiceRepo.SendWorkerError(task, err)
			if repoErr == nil {
				_ = this.stopProcessInstance(task.ProcessInstanceId) //error is sent --> no more retries
			}
			//retry task after lock duration, if stop fails or repoErr != nil
		} else {
			_, err = this.smartServiceRepo.SendWorkerModules(modules)
			if err != nil {
				//undo module and retry after lock duration
				this.handler.Undo(modules, err)
				log.Println("ERROR", err)
				debug.PrintStack()
			} else {
				err = this.completeTask(task.Id, outputs)
				if err != nil {
					log.Println("ERROR", err)
					debug.PrintStack()
				}
			}
		}
	}
	return false
}

func (this *Camunda) getTasks() (tasks []model.CamundaExternalTask, err error) {
	fetchRequest := model.CamundaFetchRequest{
		WorkerId: this.config.CamundaWorkerId,
		MaxTasks: this.config.CamundaFetchMaxTasks,
		Topics:   []model.CamundaTopic{{LockDuration: this.config.CamundaLockDurationInMs, Name: this.config.CamundaWorkerTopic}},
	}
	client := http.Client{Timeout: 5 * time.Second}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(fetchRequest)
	if err != nil {
		return
	}
	endpoint := this.config.CamundaUrl + "/engine-rest/external-task/fetchAndLock"
	resp, err := client.Post(endpoint, "application/json", b)
	if err != nil {
		return tasks, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		temp, err := ioutil.ReadAll(resp.Body)
		err = errors.New(fmt.Sprintln(endpoint, resp.Status, resp.StatusCode, string(temp), err))
		return tasks, err
	}
	err = json.NewDecoder(resp.Body).Decode(&tasks)
	return
}

func (this *Camunda) completeTask(taskId string, outputs map[string]interface{}) (err error) {
	log.Println("Start complete Request")
	client := http.Client{Timeout: 5 * time.Second}

	variables := map[string]model.CamundaVariable{}
	for key, value := range outputs {
		variables[key] = model.CamundaVariable{Value: value}
	}

	var completeRequest = model.CamundaCompleteRequest{WorkerId: this.config.CamundaWorkerId, Variables: variables}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(completeRequest)
	if err != nil {
		return
	}
	resp, err := client.Post(this.config.CamundaUrl+"/engine-rest/external-task/"+url.PathEscape(taskId)+"/complete", "application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	pl, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		log.Println("WARNING: unable to complete task:", resp.StatusCode, string(temp))
	} else {
		log.Println("complete camunda task: ", completeRequest, string(pl))
	}
	return
}

func (this *Camunda) stopProcessInstance(id string) (err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/process-instance/"+url.PathEscape(id)+"?skipIoMappings=true", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	}
	msg, _ := ioutil.ReadAll(resp.Body)
	err = errors.New("error on delete in engine for /engine-rest/process-instance/" + url.PathEscape(id) + ": " + resp.Status + " " + string(msg))
	return err
}
