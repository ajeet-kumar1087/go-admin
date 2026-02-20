package models

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

// AdminUser represents a person who can log in to the admin panel.
type AdminUser struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Role         string
}

func (u *AdminUser) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return err }
	u.PasswordHash = string(hash)
	return nil
}

func (u *AdminUser) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

// Session stores active login sessions.
type Session struct {
	ID        string    `gorm:"primaryKey"`
	UserID    uint      `gorm:"index"`
	ExpiresAt time.Time `gorm:"index"`
}

// Permission defines what a role can do with a resource.
type Permission struct {
	ID           uint   `gorm:"primaryKey"`
	Role         string `gorm:"index"`
	ResourceName string `gorm:"index"`
	Action       string
}

// AuditLog records every change made in the admin panel.
type AuditLog struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"index"`
	UserEmail    string
	ResourceName string    `gorm:"index"`
	RecordID     string    `gorm:"index"`
	Action       string    
	Changes      string    
	CreatedAt    time.Time `gorm:"index"`
}
