// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2019 Renesas Inc.
// Copyright 2019 EPAM Systems Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package permissionprovider

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "gitpct.epam.com/epmd-aepr/aos_common/api/iamanager"
	"gitpct.epam.com/epmd-aepr/aos_common/utils/cryptutils"

	"aos_vis/config"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

// PermissionProvider vis permission provider
type PermissionProvider struct {
	serverURL  string
	rootCert   string
	insecure   bool
	iamClient  pb.IAManagerPublicClient
	connection *grpc.ClientConn
}

/*******************************************************************************
 * Consts
 ******************************************************************************/

const (
	iamRequestTimeout = 30 * time.Second
)

const visFunctionalServerId = "vis"

/*******************************************************************************
 * Public
 ******************************************************************************/

// New creates new permission provider
func New(config *config.Config, insecure bool) (provider *PermissionProvider, err error) {
	provider = &PermissionProvider{serverURL: config.PermissionServerURL,
		rootCert: config.CACert, iamClient: nil, insecure: insecure, connection: nil}

	return provider, nil
}

// GetVisPermissionByToken get vis permission by token
func (provider *PermissionProvider) GetVisPermissionByToken(token string) (permissions map[string]string, err error) {
	if provider.connection == nil {
		if err = provider.connect(); err != nil {
			return permissions, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), iamRequestTimeout)
	defer cancel()

	req := &pb.GetPermissionsReq{Secret: token, FunctionalServerId: visFunctionalServerId}

	response, err := provider.iamClient.GetPermissions(ctx, req)
	if err != nil {
		return permissions, err
	}

	return response.Permissions.Permissions, nil
}

// Close close connection with permission provider grpc server
func (provider *PermissionProvider) Close() {
	if provider.connection != nil {
		provider.connection.Close()
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (provider *PermissionProvider) connect() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), iamRequestTimeout)
	defer cancel()

	var secureOpt grpc.DialOption

	if provider.insecure {
		secureOpt = grpc.WithInsecure()
	} else {
		tlsConfig, err := cryptutils.GetClientTLSConfig(provider.rootCert)
		if err != nil {
			return err
		}

		secureOpt = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}

	if provider.connection, err = grpc.DialContext(ctx, provider.serverURL, secureOpt, grpc.WithBlock()); err != nil {
		return err
	}

	provider.iamClient = pb.NewIAManagerPublicClient(provider.connection)

	return nil
}