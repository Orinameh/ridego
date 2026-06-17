package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/ridego/proto/location"
	"github.com/ridego/services/location/internal/models"
	"github.com/ridego/services/location/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedLocationServiceServer
	svc service.LocationService
}

func New(svc service.LocationService) *Server { return &Server{svc: svc} }

func (s *Server) NearbyDrivers(ctx context.Context, req *pb.NearbyRequest) (*pb.NearbyResponse, error) {
	if req.RadiusKm <= 0 {
		req.RadiusKm = 5
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	drivers, err := s.svc.NearbyDrivers(ctx, models.NearbyRequest{
		Lat:      req.Lat,
		Lng:      req.Lng,
		RadiusKm: req.RadiusKm,
		Limit:    int(req.Limit),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "nearby drivers: %v", err)
	}

	resp := &pb.NearbyResponse{}
	for _, d := range drivers {
		resp.Drivers = append(resp.Drivers, &pb.DriverPosition{
			DriverId:   d.DriverID,
			Lat:        d.Lat,
			Lng:        d.Lng,
			DistanceKm: d.DistanceKm,
			Rating:     d.Rating,
			IsOnline:   d.IsOnline,
		})
	}
	return resp, nil
}

func (s *Server) GetDriverLocation(ctx context.Context, req *pb.DriverRequest) (*pb.DriverPosition, error) {
	loc, err := s.svc.GetDriverLocation(ctx, req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "driver %s not found", req.DriverId)
	}
	return &pb.DriverPosition{
		DriverId: loc.DriverID, Lat: loc.Lat, Lng: loc.Lng, IsOnline: true,
	}, nil
}

func (s *Server) UpdateLocation(ctx context.Context, req *pb.LocationUpdate) (*pb.Ack, error) {
	err := s.svc.UpdateLocation(ctx, models.UpdateLocationInput{
		DriverID: req.DriverId, Lat: req.Lat, Lng: req.Lng,
		Heading: req.Heading, SpeedKmh: req.SpeedKmh,
	})
	if err != nil {
		return &pb.Ack{Ok: false, Message: err.Error()}, nil
	}
	return &pb.Ack{Ok: true}, nil
}

// LoggingInterceptor logs every gRPC unary call with latency.
func LoggingInterceptor(
	ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	slog.Info("grpc",
		"method", info.FullMethod,
		"latency", time.Since(start),
		"err", err,
	)
	return resp, err
}
