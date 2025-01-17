package util

import (
	"encoding/json"
	"fmt"
)

type GenericJsonError struct {
	Payload  interface{}
	HTTPCode int
}

func (e GenericJsonError) Error() string {
	return fmt.Sprintf("%v:%d", e.Payload, e.HTTPCode)
}

type ResponseObject struct {
	Ok        bool        `json:"ok"`
	Result    interface{} `json:"result,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
	Error     string      `json:"error,omitempty"`
	// backward compatibility
	ErrorMessage string        `json:"error_message,omitempty"`
	Errors       []interface{} `json:"errors,omitempty"`
}

func MakeSuccessResponse(result ...interface{}) (r ResponseObject) {

	r = ResponseObject{
		Ok:     true,
		Result: nil,
	}

	if len(result) > 0 {
		r.Result = result[0]

	}

	return
}

func MakeErrorResponse(err error) ResponseObject {

	resp := ResponseObject{
		Ok: false,
	}

	if _, ok := err.(json.Marshaler); ok {
		resp.Errors = append(resp.Errors, err)
	}

	resp.Error = err.Error()
	resp.ErrorMessage = err.Error()

	return resp
}

func ErrorResponse(message string, errorCode ...string) interface{} {
	response := ResponseObject{
		Ok:    false,
		Error: message,
	}

	if len(errorCode) > 0 {
		response.ErrorCode = errorCode[0]
	}

	return response
}
