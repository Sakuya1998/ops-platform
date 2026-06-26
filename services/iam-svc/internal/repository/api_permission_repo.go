package repository

import (
	"strings"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/model"
	"gorm.io/gorm"
)

type APIPermissionRepository struct {
	db *gorm.DB
}

func NewAPIPermissionRepository(db *gorm.DB) *APIPermissionRepository {
	return &APIPermissionRepository{db: db}
}

func (r *APIPermissionRepository) GetByRoute(method, pathPattern string) (*model.APIPermission, error) {
	var permission model.APIPermission
	err := r.db.First(&permission, "method = ? AND path_pattern = ? AND enabled = ?", strings.ToUpper(method), pathPattern, true).Error
	return &permission, err
}

func (r *APIPermissionRepository) List() ([]model.APIPermission, error) {
	var permissions []model.APIPermission
	err := r.db.Order("method ASC,path_pattern ASC").Find(&permissions).Error
	return permissions, err
}

func (r *APIPermissionRepository) Create(permission *model.APIPermission) error {
	permission.Method = strings.ToUpper(permission.Method)
	return r.db.Create(permission).Error
}

func (r *APIPermissionRepository) Update(permission *model.APIPermission) error {
	permission.Method = strings.ToUpper(permission.Method)
	return r.db.Save(permission).Error
}

func (r *APIPermissionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.APIPermission{}, "id = ?", id).Error
}
