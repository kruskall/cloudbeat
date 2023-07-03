// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package providers

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/cloudbeat/config"
	"github.com/elastic/cloudbeat/resources/providers/awslib"
	"github.com/elastic/cloudbeat/resources/utils/testhelper"
)

type ClusterProviderTestSuite struct {
	suite.Suite
}

func TestClusterProviderTestSuite(t *testing.T) {
	s := new(ClusterProviderTestSuite)

	suite.Run(t, s)
}

func (s *ClusterProviderTestSuite) TestGetClusterName() {
	tests := []struct {
		config              config.Config
		vanillaClusterName  string
		eksClusterName      string
		expectedClusterName string
	}{
		{
			config.Config{
				Benchmark:  config.CIS_K8S,
				KubeConfig: "",
			},
			"vanilla-cluster",
			"eks-cluster",
			"vanilla-cluster",
		},
		{
			config.Config{
				Benchmark: config.CIS_EKS,
			},
			"vanilla-cluster",
			"eks-cluster",
			"eks-cluster",
		},
		{
			config.Config{
				Benchmark: config.CIS_AWS,
			},
			"",
			"",
			"",
		},
	}

	for _, test := range tests {
		metaDataProvider := &awslib.MockMetadataProvider{}
		metaDataProvider.EXPECT().GetMetadata(mock.Anything, mock.Anything).
			Return(&awslib.Ec2Metadata{}, nil)

		kubernetesClusterProvider := &MockKubernetesClusterNameProviderApi{}
		kubernetesClusterProvider.EXPECT().GetClusterName(mock.Anything, mock.Anything).
			Return(test.vanillaClusterName, nil)

		eksClusterNameProviderMock := &awslib.MockClusterNameProvider{}
		eksClusterNameProviderMock.EXPECT().GetClusterName(mock.Anything, mock.Anything, mock.Anything).
			Return(test.eksClusterName, nil)

		configProviderMock := &awslib.MockConfigProviderAPI{}
		configProviderMock.EXPECT().InitializeAWSConfig(mock.Anything, mock.Anything).
			Return(&awssdk.Config{}, nil)

		kubeClient := k8sfake.NewSimpleClientset()
		clusterProvider := ClusterNameProvider{
			KubernetesClusterNameProvider: kubernetesClusterProvider,
			EKSClusterNameProvider:        eksClusterNameProviderMock,
			EKSMetadataProvider:           metaDataProvider,
			KubeClient:                    kubeClient,
			AwsConfigProvider:             configProviderMock,
		}

		ctx := context.Background()
		clusterName, err := clusterProvider.GetClusterName(ctx, &test.config, testhelper.NewLogger(s.T()))

		s.NoError(err)
		s.Equal(test.expectedClusterName, clusterName)
	}
}

func (s *ClusterProviderTestSuite) TestGetClusterNameNoValidIntegrationType() {
	clusterProvider := ClusterNameProvider{}
	ctx := context.Background()
	cfg := config.Config{
		Benchmark: "invalid-type",
		CloudConfig: config.CloudConfig{
			AwsCred: aws.ConfigAWS{},
		},
	}

	s.Panics(func() { _, _ = clusterProvider.GetClusterName(ctx, &cfg, nil) })
}
