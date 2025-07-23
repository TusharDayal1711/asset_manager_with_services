package handler

import (
	"asset/database"
	"github.com/google/uuid"
	"log"
)

func CreateFirstAdmin() bool {
	const adminEmail = "systemadmin@remotestate.com"
	const adminUsername = "System Admin"
	const Role = "admin"
	const Type = "full_time"

	var isExist uuid.UUID
	err := database.DB.Get(&isExist, `
		SELECT id FROM users 
		WHERE email = $1 AND archived_at IS NULL
	`, adminEmail)
	if err == nil {
		log.Println("user id already exist", isExist)
		return false
	}

	tx, err := database.DB.Beginx()
	if err != nil {
		log.Println("transaction failed", err)
		return false
	}

	defer func() {
		if p := recover(); p != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var adminID uuid.UUID
	err = tx.Get(&adminID, `
		INSERT INTO users (username, email)
		VALUES ($1, $2)
		RETURNING id
	`, adminUsername, adminEmail)
	if err != nil {
		log.Println("failed to create new admin", err)
		return false
	}

	_, err = tx.Exec(`
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $2)
	`, Role, adminID)
	if err != nil {
		log.Println("failed to assign role", err)
		return false
	}

	_, err = tx.Exec(`
		INSERT INTO user_type (type, user_id, created_by)
		VALUES ($1, $2, $2)
	`, Type, adminID)
	if err != nil {
		log.Println("failed to assign user type", err)
		return false
	}
	log.Println("admin created", adminID)
	return true
}
