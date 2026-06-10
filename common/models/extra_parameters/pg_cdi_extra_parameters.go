// Copyright 2026 NTT, Inc.
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

package extra_parameters

type PgCDIExtraParameters struct {
	CDIGuest     		string `json:"cdi_guest" validate:"required,min=1,max=128"`
	CDIUser     		string `json:"cdi_user" validate:"required,min=1,max=32"`
	CDIPassword  		string `json:"cdi_password" validate:"required,min=1,max=32"`
	CDIMgrGuestUser     string `json:"cdimgr_guest_user"`
	CDIMgrGuestPassword string `json:"cdimgr_guest_password"`
	CDIMgrHostPassword  string `json:"cdimgr_host_password"`
	DirectorPassword    string `json:"director_password"`
}
