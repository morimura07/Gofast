package skeleton

import (
	"context"
	"errors"
	"fmt"
	proto "gofast/gen/proto/v1"
	"gofast/pkg"
	"gofast/service-core/domain/skeleton"
	"gofast/service-core/storage/query"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type Server struct {
	deps skeleton.Deps
}

func NewSkeletonServer(deps skeleton.Deps) *Server {
	return &Server{deps: deps}
}

func queryToProto(skeleton *query.Skeleton) *proto.Skeleton {
	return &proto.Skeleton{
		Id:      skeleton.ID.String(),
		Created: skeleton.Created.Format(time.RFC3339),
		Updated: skeleton.Updated.Format(time.RFC3339),
		Name:    skeleton.Name,
		Age:     skeleton.Age,
		Death:   skeleton.Death.Format(time.RFC3339),
		Zombie:  skeleton.Zombie,
	}
}

func (s *Server) GetAllSkeletons(
	ctx context.Context,
	_ *connect.Request[proto.GetAllSkeletonsRequest],
	stream *connect.ServerStream[proto.GetAllSkeletonsResponse],
) error {
	return s.GetAllSkeletonsLogic(ctx, stream.Send)
}

func (s *Server) GetAllSkeletonsLogic(ctx context.Context, send func(*proto.GetAllSkeletonsResponse) error) error {
	processor := func(_ context.Context, skel *query.Skeleton) error {
		if ctx.Err() != nil {
			return pkg.InternalError{Err: ctx.Err()}
		}
		return send(&proto.GetAllSkeletonsResponse{Skeleton: queryToProto(skel)})
	}
	err := skeleton.GetAllSkeletons(ctx, &s.deps, processor)
	if err != nil {
		return fmt.Errorf("error getting all skeletons: %w", err)
	}
	return nil
}

func (s *Server) GetSkeletonByID(
	ctx context.Context,
	req *connect.Request[proto.GetSkeletonByIDRequest],
) (*connect.Response[proto.GetSkeletonByIDResponse], error) {
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, pkg.BadRequestError{Err: err}
	}

	skel, err := skeleton.GetSkeletonByID(ctx, &s.deps, id)
	if err != nil {
		return nil, fmt.Errorf("error getting skeleton by id: %w", err)
	}

	return connect.NewResponse(&proto.GetSkeletonByIDResponse{Skeleton: queryToProto(skel)}), nil
}

func (s *Server) CreateSkeleton(
	ctx context.Context,
	req *connect.Request[proto.CreateSkeletonRequest],
) (*connect.Response[proto.CreateSkeletonResponse], error) {
	if req.Msg.GetSkeleton() == nil {
		return nil, pkg.BadRequestError{Err: errors.New("skeleton is nil")}
	}

	created, err := skeleton.CreateSkeleton(ctx, &s.deps, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("error creating skeleton: %w", err)
	}

	return connect.NewResponse(&proto.CreateSkeletonResponse{Skeleton: queryToProto(created)}), nil
}

func (s *Server) EditSkeleton(
	ctx context.Context,
	req *connect.Request[proto.EditSkeletonRequest],
) (*connect.Response[proto.EditSkeletonResponse], error) {
	if req.Msg.GetSkeleton() == nil {
		return nil, pkg.BadRequestError{Err: errors.New("skeleton is nil")}
	}

	edited, err := skeleton.EditSkeleton(ctx, &s.deps, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("error editing skeleton: %w", err)
	}

	return connect.NewResponse(&proto.EditSkeletonResponse{Skeleton: queryToProto(edited)}), nil
}

func (s *Server) RemoveSkeleton(
	ctx context.Context,
	req *connect.Request[proto.RemoveSkeletonRequest],
) (*connect.Response[proto.RemoveSkeletonResponse], error) {
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, pkg.BadRequestError{Err: err}
	}

	err = skeleton.RemoveSkeleton(ctx, &s.deps, id)
	if err != nil {
		return nil, fmt.Errorf("error removing skeleton: %w", err)
	}

	return connect.NewResponse(&proto.RemoveSkeletonResponse{}), nil
}
