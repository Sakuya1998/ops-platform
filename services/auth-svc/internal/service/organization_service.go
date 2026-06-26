package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
)

func (s *AuthService) ListOrganizations() ([]model.Organization, error) {
	return s.orgRepo.List()
}

func (s *AuthService) CreateOrganization(name, code, description string) (*model.Organization, error) {
	org := &model.Organization{Name: name, Code: code, Description: description, Status: "active"}
	if err := s.orgRepo.Create(org); err != nil {
		return nil, err
	}
	s.publishEvent("org.created", "", org.ID.String(), code, "create", "Organization created")
	return org, nil
}

func (s *AuthService) UpdateOrganization(id, name, description string) (*model.Organization, error) {
	oid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid org id")
	}
	org, err := s.orgRepo.GetByID(oid)
	if err != nil {
		return nil, err
	}
	org.Name = name
	org.Description = description
	if err := s.orgRepo.Update(org); err != nil {
		return nil, err
	}
	s.publishEvent("org.updated", "", org.ID.String(), org.Code, "update", "Organization updated")
	return org, nil
}

func (s *AuthService) resolveOrgID(orgCode string) uuid.UUID {
	if orgCode == "" || orgCode == "default" {
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	if id, err := uuid.Parse(orgCode); err == nil {
		return id
	}
	if s.orgRepo != nil {
		if org, err := s.orgRepo.GetByCode(orgCode); err == nil {
			return org.ID
		}
	}
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}
