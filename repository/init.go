package repository

import "gorm.io/gorm"

var (
	UserRepo   IUserRepository
	OffsetRepo IOffsetRepository
)

func InitRepositories(db *gorm.DB) {
	UserRepo = UserRepositoryImpl(db)
	OffsetRepo = OffsetRepositoryImpl(db)
}
