// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type WebhookServer struct {
	ServiceAddress string `yaml:"service_address" env:"WEBHOOK_SERVER_SERVICE_ADDRESS" required:"true" env-description:"Webhook Server Service Address"`
	TLSSecretFile  string `yaml:"tls_secret_file" env:"WEBHOOK_SERVER_TLS_SECRET_FILE" required:"true"`
	CACert         []byte `yaml:"ca_cert" env:"WEBHOOK_SERVER_CA_CERT" required:"true"`
}
