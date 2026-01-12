package transport

import (
	"context"
	"errors"
	"gofast/pkg"

	"connectrpc.com/connect"
)

func NewErrorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				var validationErrors pkg.ValidationErrors
				var notFoundError pkg.NotFoundError
				var badRequestError pkg.BadRequestError
				var unauthorizedError pkg.UnauthorizedError
				var forbiddenError pkg.ForbiddenError
				var internalError pkg.InternalError

				switch {
				case errors.As(err, &validationErrors):
					return nil, connect.NewError(connect.CodeInvalidArgument, validationErrors)
				case errors.As(err, &notFoundError):
					return nil, connect.NewError(connect.CodeNotFound, errors.New("not found"))
				case errors.As(err, &badRequestError):
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
				case errors.As(err, &unauthorizedError):
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthorized"))
				case errors.As(err, &forbiddenError):
					return nil, connect.NewError(connect.CodePermissionDenied, forbiddenError.Err)
				case errors.As(err, &internalError):
					return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
				}
			}
			return resp, err
		}
	}
}
