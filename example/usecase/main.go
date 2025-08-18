package main

import (
	"context"
	"time"

	dbgo "github.com/adnvilla/db-go"
	"github.com/adnvilla/logger-go"
)

func main() {
	ctx := context.Background()

	db := dbgo.GetConnection(dbgo.Config{
		PrimaryDSN: "host=localhost user=youruser password=yourpassword dbname=yourdb port=5432 sslmode=disable",
	})
	if db.Error != nil {
		panic(db.Error)
	}
	ctx = dbgo.SetFromContext(ctx, db.Instance)

	userRepo := NewUserRepository()
	userService := NewUserService(userRepo)

	user, err := userService.CreateUserAndLog(ctx, "Juan", "adnvilla@example.com")
	if err != nil {
		logger.Error(ctx, "failed to create user: %v", err)
		return
	}

	logger.Info(ctx, "User created successfully: %+v", user)
}

type User struct {
	ID        uint
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uint) (*User, error)
}

type UserService struct {
	userRepo UserRepository
}

type gormUserRepository struct {
}

func NewUserRepository() UserRepository {
	return &gormUserRepository{}
}

func (r *gormUserRepository) Save(ctx context.Context, user *User) error {
	db := dbgo.GetFromContext(ctx)
	return db.Save(user).Error
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	var user User
	db := dbgo.GetFromContext(ctx)
	result := db.First(&user, id)
	return &user, result.Error
}

func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) CreateUserAndLog(ctx context.Context, name, email string) (*User, error) {
	user := &User{Name: name, Email: email}
	err := dbgo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Save(txCtx, user); err != nil {
			return err
		}

		return nil
	})

	return user, err
}
