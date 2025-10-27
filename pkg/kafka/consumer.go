/*
 * Copyright 2019 InfAI (CC SES)
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

package kafka

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, logger *slog.Logger, kafkaUrl string, consumerGroup string, topic string, listener func(delivery []byte, t time.Time) error) error {
	broker, err := GetBroker(kafkaUrl)
	if err != nil {
		logger.Error("unable to get broker list", "error", err)
		return err
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		CommitInterval: 0, //synchronous commits
		Brokers:        broker,
		GroupID:        consumerGroup,
		Topic:          topic,
		MaxWait:        1 * time.Second,
		Logger:         log.New(io.Discard, "", 0),
		ErrorLogger:    log.New(io.Discard, "", 0),
		StartOffset:    kafka.LastOffset,
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer r.Close()
		defer logger.Info("close consumer for topic " + topic)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := r.FetchMessage(ctx)
				if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					logger.Error("error while consuming topic", "error", err)
					log.Fatal("ERROR: while consuming topic ", topic, err)
					return
				}

				err = retry(func() error {
					return listener(m.Value, m.Time)
				}, func(n int64) time.Duration {
					return time.Duration(n) * time.Second
				}, 10*time.Minute)

				if err != nil {
					logger.Error("unable to handle message (no commit)", "error", err)
					log.Fatal("ERROR: unable to handle message (no commit)", err)
				} else {
					err = r.CommitMessages(ctx, m)
					if err != nil {
						logger.Error("error while committing consumption of topic "+topic, "error", err)
						log.Fatal("ERROR: while committing consumption ", topic, err)
						return
					}
				}
			}
		}
	}()
	return nil
}

func retry(f func() error, waitProvider func(n int64) time.Duration, timeout time.Duration) (err error) {
	err = errors.New("initial")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
		if err != nil {
			wait := waitProvider(i)
			if time.Since(start)+wait < timeout {
				time.Sleep(wait)
			} else {
				return err
			}
		}
	}
	return err
}
