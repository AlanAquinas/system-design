package main

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"encoding/json"
)

var db *gorm.DB
var jwtSecret = []byte("09d25e094faa6ca2556c818166b7a9563b93f7099f6f0f4caa6cf63b88e8d3e7")

// User model
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	FullName string
	Email    string `gorm:"unique"`
	Password string `gorm:"not null"`
	Disabled bool   `gorm:"default:false"`
	Scopes   string `gorm:"type:text"`
}

// UserLogin represents login payload
type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse represents token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// InitDB initializes database connection
//
//	func InitDB() {
//		dsn := "host=pgbouncer user=fastapi_user password=fastapi_pass dbname=fastapi_db port=5432 sslmode=disable"
//		var err error
//		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
//		if err != nil {
//			log.Fatal("Failed to connect to database", err)
//		}
//		log.Println("Database connected")
//		db.AutoMigrate(&User{})
//	}
func InitDB() {
	dsn := "host=pgbouncer user=fastapi_user password=fastapi_pass dbname=fastapi_db port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}
	log.Println("Database connected")

	// Get the underlying database connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance", err)
	}

	// Set the maximum number of open connections
	sqlDB.SetMaxOpenConns(100) // You can adjust this number based on your needs

	// Set the maximum number of idle connections
	sqlDB.SetMaxIdleConns(10) // You can adjust this number based on your needs

	// Set the maximum connection lifetime (in seconds)
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // You can adjust this based on your needs

	// AutoMigrate will create or update the database schema
	db.AutoMigrate(&User{})
}

// HashPassword hashes a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares password and hash
func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CreateToken generates JWT token
//
//	func CreateToken(username string, scopes string) (string, error) {
//		expirationTime := time.Now().Add(2 * time.Hour)
//		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
//			"sub":    username,
//			"scopes": scopes,
//			"exp":    expirationTime.Unix(),
//		})
//		return token.SignedString(jwtSecret)
//	}
func CreateToken(username string, scopes string) (string, error) {
	expirationTime := time.Now().Add(2 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    username,
		"scopes": scopes,
		"exp":    expirationTime.Unix(),
	})
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, err
	}

	return claims, nil
}

func main() {
	InitDB()
	app := fiber.New()
	app.Use(logger.New())

	app.Post("/token", func(c *fiber.Ctx) error {
		var input UserLogin
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		var user User
		result := db.Where("username = ?", input.Username).First(&user)
		if result.Error != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		if !CheckPasswordHash(input.Password, user.Password) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
		}

		scopesJSON, err := json.Marshal(user.Scopes)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not process scopes"})
		}

		token, err := CreateToken(user.Username, string(scopesJSON))

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not generate token"})
		}

		return c.JSON(TokenResponse{AccessToken: token, TokenType: "bearer"})
	})

	app.Post("/check", func(c *fiber.Ctx) error {
		// Get the Authorization header from the request
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header is required"})
		}

		// Extract the token from the Authorization header (format: Bearer <token>)
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}

		// Validate the token
		claims, err := ValidateToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Check if the token is expired
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token has expired"})
			}
		}

		// Return the token's scopes
		scopes := claims["scopes"].(string)
		return c.JSON(fiber.Map{"scopes": scopes})
	})

	// app.Post("/users", func(c *fiber.Ctx) error {
	// var user User
	// if err := c.BodyParser(&user); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	// }

	// hashedPassword, err := HashPassword(user.Password)
	// if err != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	// }
	// user.Password = hashedPassword

	// result := db.Create(&user)
	// if result.Error != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user"})
	// }

	// return c.Status(fiber.StatusCreated).JSON(user)
	// var user User
	// if err := c.BodyParser(&user); err != nil {
	// 	log.Println("Error parsing request body:", err)
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	// }

	// hashedPassword, err := HashPassword(user.Password)
	// if err != nil {
	// 	log.Println("Error hashing password:", err)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	// }
	// user.Password = hashedPassword

	// result := db.Create(&user)
	// if result.Error != nil {
	// 	log.Println("Error saving user to database:", result.Error)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user"})
	// }

	// return c.Status(fiber.StatusCreated).JSON(user)

	// })
	app.Post("/users", func(c *fiber.Ctx) error {
		var input struct {
			Username string   `json:"username"`
			Password string   `json:"password"`
			FullName string   `json:"full_name"`
			Email    string   `json:"email"`
			Disabled bool     `json:"disabled"`
			Scopes   []string `json:"scopes"`
		}

		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
		}

		hashedPassword, err := HashPassword(input.Password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
		}

		user := User{
			Username: input.Username,
			Password: hashedPassword,
			FullName: input.FullName,
			Email:    input.Email,
			Disabled: input.Disabled,
			Scopes:   strings.Join(input.Scopes, ","), // Convert slice to string
		}

		if result := db.Create(&user); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user"})
		}

		return c.Status(fiber.StatusCreated).JSON(user)
	})

	log.Fatal(app.Listen(":8000"))
}
