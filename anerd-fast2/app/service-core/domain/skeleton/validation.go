package skeleton

import (
	"gofast/pkg"
	"gofast/pkg/str"
	"gofast/service-core/storage/query"
	"strconv"

	proto "gofast/gen/proto/v1"

	"github.com/google/uuid"
)

func ValidateAndBuildInsertParams(userID uuid.UUID, skeleton *proto.Skeleton) (*query.InsertSkeletonParams, []pkg.ValidationError) {
	errors := make([]pkg.ValidationError, 0)
	if skeleton.GetName() == "" {
		errors = append(errors, pkg.ValidationError{Field: "name", Tag: "required", Message: "Name is required"})
	}
	if skeleton.GetName() != "" && len(skeleton.GetName()) < 3 {
		errors = append(errors, pkg.ValidationError{Field: "name", Tag: "minlength", Message: "Name must be at least 3 characters long"})
	}
	ageFloat, err := strconv.ParseFloat(skeleton.GetAge(), 64)
	if err != nil {
		errors = append(errors, pkg.ValidationError{Field: "age", Tag: "number", Message: "Age must be a number"})
	} else if ageFloat < 1 {
		errors = append(errors, pkg.ValidationError{Field: "age", Tag: "gte", Message: "Age must be greater than or equal to 1"})
	}
	death, err := str.ParseDate(skeleton.GetDeath())
	if err != nil {
		errors = append(errors, pkg.ValidationError{Field: "death", Tag: "required", Message: "Death date is required and must be in YYYY-MM-DD or RFC3339 format"})
	}
	if len(errors) > 0 {
		return nil, errors
	}

	return &query.InsertSkeletonParams{
		UserID: userID,
		Name:   skeleton.GetName(),
		Age:    skeleton.GetAge(),
		Death:  death,
		Zombie: skeleton.GetZombie(),
	}, nil
}

func ValidateAndBuildUpdateParams(userID uuid.UUID, skeleton *proto.Skeleton) (*query.UpdateSkeletonParams, []pkg.ValidationError) {
	errors := make([]pkg.ValidationError, 0)
	id, err := uuid.Parse(skeleton.GetId())
	if err != nil {
		errors = append(errors, pkg.ValidationError{Field: "id", Tag: "uuid", Message: "ID must be a valid UUID"})
	}
	if id == uuid.Nil {
		errors = append(errors, pkg.ValidationError{Field: "id", Tag: "required", Message: "ID is required"})
	}
	if skeleton.GetName() == "" {
		errors = append(errors, pkg.ValidationError{Field: "name", Tag: "required", Message: "Name is required"})
	}
	if skeleton.GetName() != "" && len(skeleton.GetName()) < 3 {
		errors = append(errors, pkg.ValidationError{Field: "name", Tag: "minlength", Message: "Name must be at least 3 characters long"})
	}
	ageFloat, err := strconv.ParseFloat(skeleton.GetAge(), 64)
	if err != nil {
		errors = append(errors, pkg.ValidationError{Field: "age", Tag: "number", Message: "Age must be a number"})
	} else if ageFloat < 1 {
		errors = append(errors, pkg.ValidationError{Field: "age", Tag: "gte", Message: "Age must be greater than or equal to 1"})
	}
	death, err := str.ParseDate(skeleton.GetDeath())
	if err != nil {
		errors = append(errors, pkg.ValidationError{Field: "death", Tag: "required", Message: "Death date is required and must be in YYYY-MM-DD or RFC3339 format"})
	}
	if len(errors) > 0 {
		return nil, errors
	}

	return &query.UpdateSkeletonParams{
		ID:     id,
		UserID: userID,
		Name:   skeleton.GetName(),
		Age:    skeleton.GetAge(),
		Death:  death,
		Zombie: skeleton.GetZombie(),
	}, nil
}
