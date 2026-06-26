package repository

import (
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/model"
	"gorm.io/gorm"
)

type PolicyRepository struct {
	db *gorm.DB
}

func NewPolicyRepository(db *gorm.DB) *PolicyRepository {
	return &PolicyRepository{db: db}
}

func (r *PolicyRepository) CreateResource(resource *model.Resource) error {
	return r.db.Create(resource).Error
}

func (r *PolicyRepository) CreatePolicy(policy *model.Policy) error {
	return r.db.Create(policy).Error
}

func (r *PolicyRepository) BindPolicy(binding *model.PolicyBinding) error {
	return r.db.Create(binding).Error
}
