package grpc

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/service"
	notifyv1 "github.com/Sakuya1998/ops-platform/pkg/proto/notify/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	notifyv1.UnimplementedNotifyServiceServer
	channelRepo *repository.ChannelRepository
	tmplRepo    *repository.TemplateRepository
	logRepo     *repository.LogRepository
	svc         *service.NotifyService
}

func NewServer(cr *repository.ChannelRepository, tr *repository.TemplateRepository, lr *repository.LogRepository, svc *service.NotifyService) *Server {
	return &Server{channelRepo: cr, tmplRepo: tr, logRepo: lr, svc: svc}
}

func (s *Server) ListChannels(ctx context.Context, req *notifyv1.ListChannelsRequest) (*notifyv1.ListChannelsResponse, error) {
	channels, err := s.channelRepo.ListByOrg(parseUUID(req.OrgId))
	if err != nil {
		return nil, err
	}
	resp := &notifyv1.ListChannelsResponse{Channels: make([]*notifyv1.NotificationChannel, 0, len(channels))}
	for _, channel := range channels {
		resp.Channels = append(resp.Channels, mapChannel(channel))
	}
	return resp, nil
}

func (s *Server) CreateChannel(ctx context.Context, req *notifyv1.CreateChannelRequest) (*notifyv1.NotificationChannel, error) {
	channel := &model.NotificationChannel{
		OrgID:       parseUUID(req.OrgId),
		Name:        req.Name,
		ChannelType: req.ChannelType,
		Config:      req.ConfigJson,
		IsEnabled:   req.IsEnabled,
	}
	if err := s.channelRepo.Create(channel); err != nil {
		return nil, err
	}
	return mapChannel(*channel), nil
}

func (s *Server) UpdateChannel(ctx context.Context, req *notifyv1.UpdateChannelRequest) (*notifyv1.NotificationChannel, error) {
	channel, err := s.channelRepo.GetByID(parseUUID(req.Id))
	if err != nil {
		return nil, err
	}
	channel.Name = req.Name
	channel.ChannelType = req.ChannelType
	channel.Config = req.ConfigJson
	channel.IsEnabled = req.IsEnabled
	if err := s.channelRepo.Update(channel); err != nil {
		return nil, err
	}
	return mapChannel(*channel), nil
}

func (s *Server) DeleteChannel(ctx context.Context, req *notifyv1.DeleteChannelRequest) (*notifyv1.Empty, error) {
	if err := s.channelRepo.Delete(parseUUID(req.Id)); err != nil {
		return nil, err
	}
	return &notifyv1.Empty{}, nil
}

func (s *Server) ListTemplates(ctx context.Context, req *notifyv1.ListTemplatesRequest) (*notifyv1.ListTemplatesResponse, error) {
	templates, err := s.tmplRepo.List()
	if err != nil {
		return nil, err
	}
	resp := &notifyv1.ListTemplatesResponse{Templates: make([]*notifyv1.NotificationTemplate, 0, len(templates))}
	for _, tmpl := range templates {
		resp.Templates = append(resp.Templates, mapTemplate(tmpl))
	}
	return resp, nil
}

func (s *Server) CreateTemplate(ctx context.Context, req *notifyv1.CreateTemplateRequest) (*notifyv1.NotificationTemplate, error) {
	tmpl := &model.NotificationTemplate{
		ChannelType:   req.ChannelType,
		Name:          req.Name,
		TitleTemplate: req.TitleTemplate,
		BodyTemplate:  req.BodyTemplate,
	}
	if err := s.tmplRepo.Create(tmpl); err != nil {
		return nil, err
	}
	return mapTemplate(*tmpl), nil
}

func (s *Server) UpdateTemplate(ctx context.Context, req *notifyv1.UpdateTemplateRequest) (*notifyv1.NotificationTemplate, error) {
	tmpl, err := s.tmplRepo.GetByID(parseUUID(req.Id))
	if err != nil {
		return nil, err
	}
	tmpl.ChannelType = req.ChannelType
	tmpl.Name = req.Name
	tmpl.TitleTemplate = req.TitleTemplate
	tmpl.BodyTemplate = req.BodyTemplate
	if err := s.tmplRepo.Update(tmpl); err != nil {
		return nil, err
	}
	return mapTemplate(*tmpl), nil
}

func (s *Server) DeleteTemplate(ctx context.Context, req *notifyv1.DeleteTemplateRequest) (*notifyv1.Empty, error) {
	if err := s.tmplRepo.Delete(parseUUID(req.Id)); err != nil {
		return nil, err
	}
	return &notifyv1.Empty{}, nil
}

func (s *Server) ListNotificationLogs(ctx context.Context, req *notifyv1.ListNotificationLogsRequest) (*notifyv1.ListNotificationLogsResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	logs, total, err := s.logRepo.List(req.EventType, req.Status, req.StartTime, req.EndTime, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	resp := &notifyv1.ListNotificationLogsResponse{Logs: make([]*notifyv1.NotificationLog, 0, len(logs)), Total: total}
	for _, log := range logs {
		resp.Logs = append(resp.Logs, mapLog(log))
	}
	return resp, nil
}

func (s *Server) SendNotification(ctx context.Context, req *notifyv1.SendNotificationRequest) (*notifyv1.SendNotificationResponse, error) {
	cfgMap := map[string]interface{}{}
	if req.VariablesJson != "" {
		_ = json.Unmarshal([]byte(req.VariablesJson), &cfgMap)
	}
	cfgMap["channel_type"] = req.ChannelType
	sendErr := s.svc.Send(cfgMap, req.Recipient, req.Title, req.Body)

	logEntry := &model.NotificationLog{
		EventType: req.EventType,
		Recipient: req.Recipient,
		Title:     req.Title,
		Status:    "success",
	}
	if req.ChannelId != "" {
		channelID := parseUUID(req.ChannelId)
		if channelID != uuid.Nil {
			logEntry.ChannelID = &channelID
		}
	}
	if sendErr != nil {
		logEntry.Status = "failed"
		logEntry.ErrorMsg = sendErr.Error()
	}
	if err := s.logRepo.Create(logEntry); err != nil {
		return nil, err
	}
	return &notifyv1.SendNotificationResponse{
		LogId:    logEntry.ID.String(),
		Status:   logEntry.Status,
		ErrorMsg: logEntry.ErrorMsg,
	}, nil
}

func mapChannel(channel model.NotificationChannel) *notifyv1.NotificationChannel {
	return &notifyv1.NotificationChannel{
		Id:          channel.ID.String(),
		OrgId:       channel.OrgID.String(),
		Name:        channel.Name,
		ChannelType: channel.ChannelType,
		ConfigJson:  channel.Config,
		IsEnabled:   channel.IsEnabled,
		CreatedAt:   timestamppb.New(channel.CreatedAt),
		UpdatedAt:   timestamppb.New(channel.UpdatedAt),
	}
}

func mapTemplate(tmpl model.NotificationTemplate) *notifyv1.NotificationTemplate {
	return &notifyv1.NotificationTemplate{
		Id:            tmpl.ID.String(),
		ChannelType:   tmpl.ChannelType,
		Name:          tmpl.Name,
		TitleTemplate: tmpl.TitleTemplate,
		BodyTemplate:  tmpl.BodyTemplate,
		CreatedAt:     timestamppb.New(tmpl.CreatedAt),
	}
}

func mapLog(log model.NotificationLog) *notifyv1.NotificationLog {
	channelID := ""
	if log.ChannelID != nil && *log.ChannelID != uuid.Nil {
		channelID = log.ChannelID.String()
	}
	return &notifyv1.NotificationLog{
		Id:        log.ID.String(),
		ChannelId: channelID,
		EventType: log.EventType,
		Recipient: log.Recipient,
		Title:     log.Title,
		Status:    log.Status,
		ErrorMsg:  log.ErrorMsg,
		CreatedAt: timestamppb.New(log.CreatedAt),
	}
}

func normalizePage(page, pageSize int32) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return int(page), int(pageSize)
}

func parseUUID(value string) uuid.UUID {
	if value == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return id
}
