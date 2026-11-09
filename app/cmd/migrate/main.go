package main

import (
	"app/internal/app/ds"
	"app/internal/app/dsn"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(
		&ds.User{},
		&ds.Telescope{},
		&ds.TelescopeLike{},
	)
	if err != nil {
		panic("cant migrate db")
	}
	if db.Migrator().HasColumn(&ds.Telescope{}, "completion_time") {
		if err := db.Migrator().DropColumn(&ds.Telescope{}, "completion_time"); err != nil {
			panic(err)
		}
	}
	if err := db.Exec("SELECT setval(pg_get_serial_sequence('users', 'id'), (SELECT MAX(id) FROM users), true)").Error; err != nil {
		panic(err)
	}
}
