package handler

import (
	"os"
	"time"

	"github.com/3n0ugh/GoFiber-RestAPI-UserAuth/server/database"
	"github.com/3n0ugh/GoFiber-RestAPI-UserAuth/server/models"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/golobby/dotenv"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username   string `json:"username" validate:"required,min=5,max=12,alphanum"`
	Password   string `json:"password" validate:"required,min=8,max=32"`
	RePassword string `json:"repassword"`
}

func Signup(c *fiber.Ctx) error {
	user := new(User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(400).JSON(err.Error())
	}
	// validate the request body
	valid := validator.New()
	err := valid.Struct(user)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	} else if user.Password != user.RePassword {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "passwords don't match")
	} else {
		// hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		// inserting into User model
		User := new(models.User)
		User.Username = user.Username
		User.Password = string(hashedPassword)
		// adding user to db
		err = database.DB.Db.Create(&User).Error
		// if user exist in db got an error
		if err != nil {
			return fiber.NewError(fiber.StatusConflict, "username already taken")
		}
		return createTokenSendResponse(c, User)
	}
}

func Login(c *fiber.Ctx) error {
	user := new(models.User)
	if err := c.BodyParser(user); err != nil {
		return c.Status(400).JSON(err.Error())
	}
	// validate the request body
	valid := validator.New()
	err := valid.Struct(user)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	} else {
		dbUser := new(models.User)
		// find the user from the database
		database.DB.Db.Where("username = ?", user.Username).Find(&dbUser)
		if dbUser.Username == "" {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "wrong username or password")
		} else {
			// checking password with hashedPassword
			err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
			if err != nil {
				return fiber.NewError(fiber.StatusUnprocessableEntity, "wrong username or password")
			} else {
				return createTokenSendResponse(c, user)
			}
		}
	}
}

func createTokenSendResponse(c *fiber.Ctx, user *models.User) error {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	// take secret_key and expire time from .envfile
	type jwtConfig struct {
		Secret             string `env:"JWT_SECRET_KEY"`
		ExpireMinutesCount int    `env:"JWT_EXPIRE_MINUTES"`
	}
	file, err := os.Open(".env")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer file.Close()
	jwtconfig := &jwtConfig{}
	err = dotenv.NewDecoder(file).Decode(jwtconfig)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	claims["username"] = user.Username
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(jwtconfig.ExpireMinutesCount)).Unix()

	// create java web token
	t, err := token.SignedString([]byte(jwtconfig.Secret))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"message": "logged in",
		"success": true,
		"data":    t,
	})
}
