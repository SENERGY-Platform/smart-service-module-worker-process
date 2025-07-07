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
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/kafka"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

func StartDoneEventHandling(ctx context.Context, wg *sync.WaitGroup, config Config, libConfig configuration.Config) error {
	if config.KafkaUrl != "" && config.KafkaUrl != "-" {
		if config.InitTopics {
			err := kafka.InitTopic(config.KafkaUrl, config.ProcessDeploymentDoneTopic)
			if err != nil {
				log.Println("ERROR: unable to create topic", err)
				return err
			}
		}
		return kafka.NewConsumer(ctx, wg, config.KafkaUrl, config.KafkaConsumerGroup, config.ProcessDeploymentDoneTopic, func(delivery []byte) error {
			msg := DoneNotification{}
			err := json.Unmarshal(delivery, &msg)
			if err != nil {
				log.Println("ERROR: unable to interpret kafka msg:", err)
				debug.PrintStack()
				return nil //ignore message
			}
			if msg.Command == "PUT" {
				eventId := deploymentIdToEventId(msg.Id)
				err = camunda.SendEventTrigger(libConfig, eventId, nil)
				if err != nil {
					log.Println("ERROR: unable to send event trigger:", err)
					debug.PrintStack()
					return err
				}
				go func() {
					time.Sleep(5 * time.Second)
					err = camunda.SendEventTrigger(libConfig, eventId, nil)
					if err != nil {
						log.Println("ERROR: unable to send event trigger:", err)
						debug.PrintStack()
					}
				}()
			}
			return nil
		})
	}
	return nil
}

func deploymentIdToEventId(id string) string {
	return "deployment_done_" + id
}

type DoneNotification struct {
	Command string `json:"command"`
	Id      string `json:"id"`
	Handler string `json:"handler"`
}
