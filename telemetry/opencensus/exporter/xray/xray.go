/*
 * Copyright (c) 2021 Huy Duc Dao
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xray

import (
	"context"
	ocAws "contrib.go.opencensus.io/exporter/aws"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/xray"
	"github.com/starvn/sonic/telemetry/opencensus"
	"time"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*ocAws.Exporter, error) {
	if cfg.Exporters.Xray == nil {
		return nil, errors.New("xray exporter disabled")
	}
	if cfg.Exporters.Xray.Version == "" {
		cfg.Exporters.Xray.Version = "Sonic-opencensus"
	}

	if !cfg.Exporters.Xray.UseEnv {
		mySession := setupAWSSession(cfg.Exporters.Xray.AccessKey, cfg.Exporters.Xray.SecretKey, cfg.Exporters.Xray.Region)
		return ocAws.NewExporter(
			ocAws.WithAPI(xray.New(mySession, aws.NewConfig().WithRegion(cfg.Exporters.Xray.Region))),
			ocAws.WithRegion(cfg.Exporters.Xray.Region),
			ocAws.WithInterval(time.Duration(cfg.ReportingPeriod)),
			ocAws.WithBufferSize(cfg.SampleRate),
			ocAws.WithVersion(cfg.Exporters.Xray.Version),
		)
	}

	return ocAws.NewExporter(
		ocAws.WithRegion(cfg.Exporters.Xray.Region),
		ocAws.WithInterval(time.Duration(cfg.ReportingPeriod)),
		ocAws.WithBufferSize(cfg.SampleRate),
		ocAws.WithVersion(cfg.Exporters.Xray.Version),
	)
}

func setupAWSSession(id, secret, region string) *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
		Region:      aws.String(region),
	}))
}
