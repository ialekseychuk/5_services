package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/example/five-services/internal/config"
	"github.com/example/five-services/pkg/random"
	pb "github.com/example/five-services/proto/gen/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service struct {
	cfg         *config.ServiceConfig
	logger      *logrus.Logger
	grpcSrv     *grpc.Server
	connections map[string]*grpc.ClientConn
	streams     map[string]pb.NeighborService_CommunicateClient
	neibers     map[string]string
	mu          sync.RWMutex
	pb.UnimplementedNeighborServiceServer
}

func NewService(cfg *config.ServiceConfig, logger *logrus.Logger) *Service {
	return &Service{
		cfg:         cfg,
		logger:      logger,
		connections: make(map[string]*grpc.ClientConn),
		streams:     make(map[string]pb.NeighborService_CommunicateClient),
		neibers:     make(map[string]string),
	}
}

func (s *Service) Run() error {
	s.logger.Info("Starting service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		s.logger.Errorf("Failed to listen: %v", err)
		return err
	}

	s.grpcSrv = grpc.NewServer()
	pb.RegisterNeighborServiceServer(s.grpcSrv, s)
	s.logger.Infof("Starting gRPC server on port %d", s.cfg.Port)

	go func() {
		if err := s.grpcSrv.Serve(lis); err != nil {
			s.logger.Errorf("Failed to serve: %v", err)
		}
	}()

	go s.discoverNeighborsLoop(ctx)

	go s.communicateWithNeighborsLoop(ctx)

	<-ctx.Done()

	s.logger.Info("Service stopped")
	return nil
}

func (s *Service) Shutdown(ctx context.Context) {
	s.logger.Info("Shutting down service...")

	if s.grpcSrv != nil {
		s.grpcSrv.GracefulStop()
	}

	// Close all connections
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		if conn != nil {
			conn.Close()
		}
	}

	s.logger.Info("Service shutdown complete")
}

func (s *Service) discoverNeighborsLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.PollingPeriod)
	defer ticker.Stop()

	// Initial discovery
	s.discoverNeighbors()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.discoverNeighbors()
		}
	}
}

func (s *Service) communicateWithNeighborsLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.MessagePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendMessagesToNeighbors()
		}
	}
}

func (s *Service) discoverNeighbors() {
	s.logger.Info("Discovering neighbors...")
	for i := 1; i <= 5; i++ {
		serviceId := fmt.Sprintf("service%d", i)
		if serviceId == s.cfg.ID {
			continue
		}

		dockerHost := fmt.Sprintf("%s:%d", serviceId, 5001)
		conn, err := net.DialTimeout("tcp", dockerHost, time.Second)
		if err != nil {
			s.logger.Debugf("Service %s not available at %s: %v", serviceId, dockerHost, err)

			s.mu.Lock()
			if _, exists := s.neibers[serviceId]; exists {
				s.logger.Infof("Service %s disconnected", serviceId)
				delete(s.neibers, serviceId)
				if conn, exists := s.connections[serviceId]; exists {
					conn.Close()
					delete(s.connections, serviceId)
				}
				delete(s.streams, serviceId)
			}
			s.mu.Unlock()
			continue
		}
		
		conn.Close()

		s.mu.Lock()
		if _, exists := s.neibers[serviceId]; !exists {
			s.logger.Infof("New service discovered: %s at %s", serviceId, dockerHost)
			s.neibers[serviceId] = dockerHost
		
			go s.conectToNeiber(serviceId, dockerHost)
		}
		s.mu.Unlock()
	}

	s.mu.RLock()
	s.logger.Infof("Discovery complete. Current neighbors: %d", len(s.neibers))
	s.mu.RUnlock()
}

func (s *Service) conectToNeiber(serviceId, dockerHost string) {
	s.mu.Lock()
	
	if _, exists := s.connections[serviceId]; exists {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.logger.Infof("Connecting to neighbor %s at %s", serviceId, dockerHost)

	conn, err := grpc.Dial(dockerHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		s.logger.Errorf("Failed to connect to neighbor %s: %v", serviceId, err)
		return
	}

	s.mu.Lock()
	s.connections[serviceId] = conn
	s.mu.Unlock()

	s.logger.Infof("Connected to service %s", serviceId)

	client := pb.NewNeighborServiceClient(conn)
	stream, err := client.Communicate(context.Background())
	if err != nil {
		s.logger.Errorf("Failed to establish stream with neighbor %s: %v", serviceId, err)
		conn.Close()
		s.mu.Lock()
		delete(s.connections, serviceId)
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.streams[serviceId] = stream
	s.mu.Unlock()
	s.logger.Infof("Established stream with neighbor %s", serviceId)

	go s.receiveMessages(serviceId, stream)
}

func (s *Service) sendMessagesToNeighbors() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.streams) == 0 {
		return
	}

	msg := random.RandomString(10)
	s.logger.Infof("Sending message %s to %d neighbors", msg, len(s.streams))

	for serviceId, stream := range s.streams {
		err := stream.Send(&pb.Message{
			Sender:    s.cfg.ID,
			Content:   msg,
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			s.logger.Errorf("Error sending message to %s: %v", serviceId, err)
			go s.handleNeighborDisconnection(serviceId)
			continue
		}

		s.logger.Infof("Sent message %s to %s", msg, serviceId)
	}
}

func (s *Service) receiveMessages(serviceID string, stream pb.NeighborService_CommunicateClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			s.logger.Errorf("Error receiving message from %s: %v", serviceID, err)
			go s.handleNeighborDisconnection(serviceID)
			return
		}

		s.logger.Infof("Received message '%s' from %s", msg.Content, serviceID)
	}
}

func (s *Service) handleNeighborDisconnection(serviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Infof("Neighbor %s disconnected", serviceID)

	delete(s.neibers, serviceID)

	if conn, ok := s.connections[serviceID]; ok {
		conn.Close()
		delete(s.connections, serviceID)
	}

	delete(s.streams, serviceID)
}

func (s *Service) Communicate(stream pb.NeighborService_CommunicateServer) error {
	s.logger.Info("Established communication stream with neighbor")

	for {
		msg, err := stream.Recv()
		if err != nil {
			s.logger.Errorf("Error receiving message: %v", err)
			return err
		}

		s.logger.Infof("Received message '%s' from %s", msg.Content, msg.Sender)

		response := &pb.Message{
			Sender:    s.cfg.ID,
			Content:   fmt.Sprintf("ACK: %s", msg.Content),
			Timestamp: time.Now().Unix(),
		}

		if err := stream.Send(response); err != nil {
			s.logger.Errorf("Error sending message: %v", err)
			return err
		}
	}
}
