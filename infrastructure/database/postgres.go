package database

import (
	"log"
	"log/slog"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func createEnums(db *gorm.DB) error {
	enumSQL := `
	DO $$ BEGIN
		CREATE TYPE project_status AS ENUM ('recruiting', 'active', 'completed');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;

	DO $$ BEGIN
		CREATE TYPE position_status AS ENUM ('open', 'closed');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;

	DO $$ BEGIN
		CREATE TYPE project_role AS ENUM ('project_lead', 'member');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;

	DO $$ BEGIN
		CREATE TYPE membership_status AS ENUM ('active', 'left', 'removed');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;

	DO $$ BEGIN
		CREATE TYPE application_status AS ENUM ('submitted', 'reviewing', 'accepted', 'rejected', 'cancelled');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;
	`

	// Eksekusi raw SQL
	return db.Exec(enumSQL).Error
}

func Connect(dbUrl string) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dbUrl,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		slog.Error("Gagal terhubung dengan supabase", "error", err)
		panic(err)
	}

	if err := createEnums(db); err != nil {
		log.Fatalf("Gagal membuat data enum: %v", err)
	}

	slog.Info("Database berhasil terhubung!")
	if err := db.AutoMigrate(
		&model.User{},
		&model.OAuthAccount{},
		&model.Project{},
		&model.Skill{},
		&model.ProjectSkill{},
		&model.ProjectPosition{},
		&model.PositionSkill{},
		&model.ProjectMember{},
		&model.Application{},
		&model.Bookmark{},
	); err != nil {
		slog.Error("Gagal melakukan migrasi database", "error", err)
		panic(err)
	}

	slog.Info("Migrasi database berhasil")
	return db
}
