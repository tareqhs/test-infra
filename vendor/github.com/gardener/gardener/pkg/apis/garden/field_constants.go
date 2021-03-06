// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package garden

// Field path constants that are specific to the internal API
// representation.
const (

	// ShootSeedNameDeprecated is the field selector path for finding
	// the Seed cluster of a garden.sapcloud.io/v1beta1 Shoot.
	// +deprecated
	ShootSeedNameDeprecated = "spec.cloud.seed"

	// ShootSeedName is the field selector path for finding
	// the Seed cluster of a core.gardener.cloud/v1alpha1 Shoot.
	ShootSeedName = "spec.seedName"

	// ShootCloudProfileName is the field selector path for finding
	// the CloudProfile name of a core.gardener.cloud/v1alpha1 Shoot.
	ShootCloudProfileName = "spec.cloudProfileName"
)
