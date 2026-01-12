package pkg

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	jsonErrors := make([]string, len(e))
	for i, ve := range e {
		jsonErrors[i] = fmt.Sprintf(`{"field": "%s", "tag": "%s", "message": "%s"}`, ve.Field, ve.Tag, ve.Message)
	}
	return fmt.Sprintf("[%s]", strings.Join(jsonErrors, ", "))
}

type InternalError struct {
	Err error
}

func (e InternalError) Error() string {
	if e.Err != nil {
		return "internal error: " + e.Err.Error()
	}
	return "internal error"
}

func (e InternalError) Unwrap() error {
	return e.Err
}

type BadRequestError struct {
	Err error
}

func (e BadRequestError) Error() string {
	if e.Err != nil {
		return "bad request: " + e.Err.Error()
	}
	return "bad request"
}

func (e BadRequestError) Unwrap() error {
	return e.Err
}

type NotFoundError struct {
	Err error
}

func (e NotFoundError) Error() string {
	if e.Err != nil {
		return "not found: " + e.Err.Error()
	}
	return "not found"
}

func (e NotFoundError) Unwrap() error {
	return e.Err
}

type UnauthorizedError struct {
	Err error
}

func (e UnauthorizedError) Error() string {
	if e.Err != nil {
		return "unauthorized: " + e.Err.Error()
	}
	return "unauthorized"
}

func (e UnauthorizedError) Unwrap() error {
	return e.Err
}

type ForbiddenError struct {
	Err error
}

func (e ForbiddenError) Error() string {
	if e.Err != nil {
		return "forbidden: " + e.Err.Error()
	}
	return "forbidden"
}

func (e ForbiddenError) Unwrap() error {
	return e.Err
}
