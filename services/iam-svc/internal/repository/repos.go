package repository

import (
	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/model"
	"gorm.io/gorm"
)

type RoleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) *RoleRepository     { return &RoleRepository{db: db} }
func (r *RoleRepository) Create(role *model.Role) error { return r.db.Create(role).Error }
func (r *RoleRepository) GetByID(id uuid.UUID) (*model.Role, error) {
	var role model.Role
	err := r.db.First(&role, "id = ?", id).Error
	return &role, err
}
func (r *RoleRepository) ListByOrg(orgID uuid.UUID) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Where("org_id = ?", orgID).Order("created_at ASC").Find(&roles).Error
	return roles, err
}
func (r *RoleRepository) GetByOrgAndCode(orgID uuid.UUID, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.First(&role, "org_id = ? AND code = ?", orgID, code).Error
	return &role, err
}
func (r *RoleRepository) Update(role *model.Role) error { return r.db.Save(role).Error }
func (r *RoleRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Role{}, "id = ?", id).Error
}
func (r *RoleRepository) AssignPermissions(roleID uuid.UUID, permIDs []uuid.UUID) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, pid := range permIDs {
		if err := tx.Create(&model.RolePermission{RoleID: roleID, PermissionID: pid}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
func (r *RoleRepository) AssignUserRoles(userID uuid.UUID, roleIDs []uuid.UUID) error {
	tx := r.db.Begin()
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, roleID := range roleIDs {
		if err := tx.Create(&model.UserRole{UserID: userID, RoleID: roleID}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (r *RoleRepository) DeleteUserRoles(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error
}

func (r *RoleRepository) GetUserRoles(userID uuid.UUID) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Raw(`SELECT r.* FROM roles r JOIN user_roles ur ON r.id = ur.role_id WHERE ur.user_id = ?`, userID).Scan(&roles).Error
	return roles, err
}

type PermissionRepository struct{ db *gorm.DB }

func NewPermissionRepository(db *gorm.DB) *PermissionRepository { return &PermissionRepository{db: db} }
func (r *PermissionRepository) ListAll() ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Order("sort ASC").Find(&perms).Error
	return perms, err
}
func (r *PermissionRepository) GetByIDs(ids []uuid.UUID) ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Where("id IN ?", ids).Find(&perms).Error
	return perms, err
}
func (r *PermissionRepository) GetRolePermissions(roleID uuid.UUID) ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Raw(`SELECT p.* FROM permissions p JOIN role_permissions rp ON p.id = rp.permission_id WHERE rp.role_id = ?`, roleID).Scan(&perms).Error
	return perms, err
}
