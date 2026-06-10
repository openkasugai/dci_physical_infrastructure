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

package utils

import (
	"fmt"
	proto "maas_module/api/proto" // import of gRPC protobuf
    common "common/api/proto"    // import of common protobuf

	"google.golang.org/grpc/codes"
)

// SeqError is a custom error type that includes a message.
type SeqError struct {
	Message string
}

func (e *SeqError) Error() string {
	return "maas controller is busy: " + e.Message
}
func (e *SeqError) ErrorDetail() *common.ErrorMessage {
	return &common.ErrorMessage{
		ErrorCode:  int32(codes.Unavailable),
		DetailCode: int32(proto.DetailCode_IF_SEQUENCE_ERROR),
		Message:    "maas controller is busy.",
	}
}

// CancelError is a custom error type
type CancelError struct {
}

func (e *CancelError) Error() string {
	return "This order cannot be cancelled due to its current status."
}
func (e *CancelError) ErrorDetail() *common.ErrorMessage {
	return &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_IF_CANCEL_UNAVAILABLE),
		Message:    e.Error(),
	}
}

// EnvError is a custom error type that includes a message.
type EnvError struct {
	Message string
}

func (e *EnvError) Error() string {
	return e.Message
}
func (e *EnvError) ErrorDetail() *common.ErrorMessage {
	return &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_MAAS_ENVIRONMENT_ERROR),
		Message:    e.Message,
	}
}

// HttpError is a custom error type that includes an HTTP status code and a message.
type HttpError struct {
	StatusCode int
	Message    string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("<%d> %s", e.StatusCode, e.Message)
}
func (e *HttpError) ErrorDetail() *common.ErrorMessage {
	return &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(e.StatusCode),
		Message:    e.Message,
	}
}

// RespError is a custom error type that includes a message.
type RespError struct {
	Message string
}

func (e *RespError) Error() string {
	return "invalid maas response: " + e.Message
}
func (e *RespError) ErrorDetail() *common.ErrorMessage {
	return &common.ErrorMessage{
		ErrorCode:  int32(codes.Internal),
		DetailCode: int32(proto.DetailCode_MAAS_RESPONSE_INVALID),
		Message:    "invalid maas response.",
	}
}
