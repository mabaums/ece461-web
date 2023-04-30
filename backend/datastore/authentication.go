package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mabaums/ece461-web/backend/models"
)

type AuthenticationToken string
type UserCredentials struct {
	Name     string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
	Password string `json:"password"`
}

var useAuthentication = true

func ExtractUserInfoFromToken(tokenString string) (string, error) {
	//fmt.Print(tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify that the signing method is HMAC and the secret matches
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		secret_key := os.Getenv("SECRET_KEY")
		return []byte(secret_key), nil
	})
	if err != nil {
		return "", err
	}

	// Extract the "user_name" claim from the token
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok1 := claims["username"].(string)
		if !ok1 {
			return "", fmt.Errorf("invalid claims")
		}
		return username, nil
	}
	return "", fmt.Errorf("invalid token")
}

/*  API Endpoints */

func CreateUser(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	var info UserCredentials
	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Print(info.IsAdmin)
	result, err := db.Exec("INSERT INTO User (name, isAdmin) VALUES (?, ?)", info.Name, info.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	userID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err = db.Exec("INSERT INTO UserAuthenticationInfo (user_id, password) VALUES (?, ?)", userID, info.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"Message": "created user"})
}

func CreateAuthToken(c *gin.Context) {
	if useAuthentication {
		db, ok := getDB(c)
		if !ok {
			return
		}

		log.Printf("CreateAuthToken: have db: %+v", db)
		//	Get authentication request from request body
		var authReq models.AuthenticationRequest
		if err := c.ShouldBindJSON(&authReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if user authentication info is correct
		// For example, verify user's password against stored hash
		var dbAuthInfo models.UserAuthenticationInfo
		err := db.QueryRow("SELECT password FROM UserAuthenticationInfo INNER JOIN User ON User.id = UserAuthenticationInfo.user_id WHERE User.name = ?", authReq.User.Name).Scan(&dbAuthInfo.Password)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Verify password
		if dbAuthInfo.Password != authReq.Secret.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"Message": "unauth"})
			return
		}
		// Verify password
		// if !verifyPassword(authReq.Secret.Password, dbAuthInfo.Password) {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		// 	return
		// }

		// Create the claims for the JWT token
		claims := jwt.MapClaims{}
		claims["username"] = authReq.User.Name
		//claims["isAdmin"] = authReq.User.IsAdmin
		claims["exp"] = time.Now().Add(time.Hour * 1).Unix() // token expires in 1 hour

		// Create the JWT token with HMAC SHA-256 signing
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Sign the token with a secret key
		err = godotenv.Load()
		if err != nil {
			log.Print("Error loading .env file")
		}
		secret_key := os.Getenv("SECRET_KEY")
		secret := []byte(secret_key)
		tokenString, err := token.SignedString(secret)
		if err != nil {
			return
		}
		c.String(http.StatusOK, tokenString)
	} else {
		c.AbortWithStatusJSON(http.StatusNotImplemented, "")
	}
}

func authenticate(c *gin.Context) bool {
	if useAuthentication {
		db, ok := getDB(c)
		if !ok {
			return false
		}
		// Authentication
		authTokenHeader := c.Request.Header.Get("X-Authorization")
		if authTokenHeader == "" {
			log.Errorf("Authenticating but did not recieve X-Authorization header")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Authentication token not found in request header"})
			return false
		}
		username, err := ExtractUserInfoFromToken(authTokenHeader)
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return false
		}
		var pass string
		log.Infof("Authenticating User: %v IP: %v", username, c.ClientIP())
		err = db.QueryRow("SELECT password FROM UserAuthenticationInfo WHERE user_id = (SELECT id FROM User WHERE name = ?)", username).Scan(&pass)
		if err != nil {
			log.Errorf("")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"description": "There is missing field(s) in the PackageData/AuthenticationToken" +
				"or it is formed improperly (e.g. Content and URL are both set), or the AuthenticationToken is invalid."})
			return false
		}
		var isAdmin bool

		err = db.QueryRow("SELECT isAdmin FROM User WHERE name = ?", username).Scan(&isAdmin)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return false
		}
		c.Set("username", username)
		c.Set("admin", isAdmin)
		log.Infof("Authenticated %v, isAdmin: %v", username, isAdmin)
		return true
	}

	log.Infof("Not authenticating, logged in %v, %v", c.ClientIP(), true)
	c.Set("username", "foo")
	c.Set("admin", true)
	return true
}
