package skeleton

import (
	"context"
	"fmt"
	"gofast/pkg"
	"gofast/pkg/auth"
	ot "gofast/pkg/otel"
	"gofast/service-core/storage/query"

	proto "gofast/gen/proto/v1"

	"github.com/google/uuid"
)

type Deps struct {
	Store *query.Queries
}

func GetAllSkeletons(ctx context.Context, d *Deps, processor func(ctx context.Context, user *query.Skeleton) error) (err error) {
	ctx, span, done := ot.StartSpan(ctx, "skeleton.service.GetAllSkeletons")
	defer func() { done(err) }()

	claims, err := auth.Authorize(ctx, span, auth.GetSkeletons)
	if err != nil {
		return pkg.ForbiddenError{Err: err}
	}

	skeletons, err := d.Store.SelectAllSkeletons(ctx, claims.ID)
	if err != nil {
		return pkg.NotFoundError{Err: err}
	}
	span.AddEvent("Skeletons selected from store")

	for _, s := range skeletons {
		err = processor(ctx, &s)
		if err != nil {
			return pkg.InternalError{Err: err}
		}
	}
	return nil
}

func GetSkeletonByID(ctx context.Context, d *Deps, id uuid.UUID) (result *query.Skeleton, err error) {
	ctx, span, done := ot.StartSpan(ctx, "skeleton.service.GetSkeletonByID")
	defer func() { done(err) }()

	claims, err := auth.Authorize(ctx, span, auth.GetSkeletons)
	if err != nil {
		return nil, pkg.ForbiddenError{Err: err}
	}

	skeleton, err := d.Store.SelectSkeletonByID(ctx, query.SelectSkeletonByIDParams{
		ID:     id,
		UserID: claims.ID,
	})
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("Skeleton selected from store")

	return &skeleton, nil
}

func CreateSkeleton(ctx context.Context, d *Deps, req *proto.CreateSkeletonRequest) (result *query.Skeleton, err error) {
	ctx, span, done := ot.StartSpan(ctx, "skeleton.service.CreateSkeleton")
	defer func() { done(err) }()

	claims, err := auth.Authorize(ctx, span, auth.CreateSkeleton)
	if err != nil {
		return nil, pkg.ForbiddenError{Err: err}
	}

	params, validation := ValidateAndBuildInsertParams(claims.ID, req.GetSkeleton())
	if validation != nil {
		return nil, fmt.Errorf("validation errors: %w", pkg.ValidationErrors(validation))
	}
	span.AddEvent("Validation successful")

	skeleton, err := d.Store.InsertSkeleton(ctx, *params)
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	span.AddEvent("Skeleton inserted into store")

	return &skeleton, nil
}

func EditSkeleton(ctx context.Context, d *Deps, req *proto.EditSkeletonRequest) (result *query.Skeleton, err error) {
	ctx, span, done := ot.StartSpan(ctx, "skeleton.service.EditSkeleton")
	defer func() { done(err) }()

	claims, err := auth.Authorize(ctx, span, auth.EditSkeleton)
	if err != nil {
		return nil, pkg.ForbiddenError{Err: err}
	}

	params, validation := ValidateAndBuildUpdateParams(claims.ID, req.GetSkeleton())
	if validation != nil {
		return nil, fmt.Errorf("validation errors: %w", pkg.ValidationErrors(validation))
	}
	span.AddEvent("Validation successful")

	skeleton, err := d.Store.UpdateSkeleton(ctx, *params)
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	span.AddEvent("Skeleton updated in store")

	return &skeleton, nil
}

func RemoveSkeleton(ctx context.Context, d *Deps, id uuid.UUID) (err error) {
	ctx, span, done := ot.StartSpan(ctx, "skeleton.service.RemoveSkeleton")
	defer func() { done(err) }()

	claims, err := auth.Authorize(ctx, span, auth.RemoveSkeleton)
	if err != nil {
		return pkg.ForbiddenError{Err: err}
	}

	err = d.Store.DeleteSkeleton(ctx, query.DeleteSkeletonParams{
		ID:     id,
		UserID: claims.ID,
	})
	if err != nil {
		return pkg.InternalError{Err: err}
	}
	span.AddEvent("Skeleton deleted from store")

	return nil
}
