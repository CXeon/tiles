package rpc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CXeon/tiles/rpc"
)

func TestResponseError_Error(t *testing.T) {
	err := &rpc.ResponseError{Code: 1001, Message: "not found", TraceID: "abc-123"}
	assert.Equal(t, "rpc error 1001: not found (trace_id=abc-123)", err.Error())
}

func TestResponseError_Error_NoTraceID(t *testing.T) {
	err := &rpc.ResponseError{Code: 500, Message: "internal error"}
	assert.Equal(t, "rpc error 500: internal error", err.Error())
}
