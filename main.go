package main

import (
	"database/sql"
	"log"
	"net/http"
	"fmt"
	"flag"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID        int64     `json:"id"`
	Firstname      string    `json:"first_name"`
	Lastname      string    `json:"last_name"`
}

type BankAccount struct {
	ID        int64     `json:"id"`
	UserID      int64    `json:"user_id"`
	AccountNumber      int64    `json:"account_number"`
	Name      string    `json:"name"`
	Balance      int64    `json:"balance"`
}

type UserService interface {
	All() ([]User, error)
	GetUserByID(id int) (*User, error)
	CreateUser(user *User) error
	UpdateUserByID(user *User) error
}

type UserServiceImp struct {
	db *sql.DB
}

func (s *UserServiceImp) All() ([]User, error) {
	stmt := "SELECT id, first_name, last_name FROM users;"
	rows, err := s.db.Query(stmt)
	if err != nil {
		return nil, err
	}
	users := []User{} // set empty slice without nil
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Firstname, &user.Lastname)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *UserServiceImp) GetUserByID(id int) (*User, error) {
	stmt := "SELECT id, first_name, last_name FROM users WHERE id = ?;"
	row := s.db.QueryRow(stmt, id)
	var user User
	err := row.Scan(&user.ID, &user.Firstname, &user.Lastname)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserServiceImp) CreateUser(user *User) error {
	stmt := "INSERT INTO users (first_name, last_name) values (?, ?);"
	res, err := s.db.Exec(stmt, user.Firstname, user.Lastname)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	user.ID = id
	return nil
}

func (s *UserServiceImp) UpdateUserByID(user *User) error {
	stmt := "UPDATE users SET first_name = ? , last_name = ? WHERE id = ?;"
	_, err := s.db.Exec(stmt, user.Firstname, user.Lastname, user.ID)
	if err != nil {
		return err
	}
	return nil
}

type AccountService interface {
}

type AccountServiceImp struct {
	db *sql.DB
}

type Server struct {
	userService    UserService
	accountService AccountService
}

func (s *Server) AllUsers(c *gin.Context) {
	users, err := s.userService.All()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("db: query error: %s", err),
		})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (s *Server) GetUserByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := s.userService.GetUserByID(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) CreateUser(c *gin.Context) {
	var user User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	if err := s.userService.CreateUser(&user); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (s *Server) UpdateUserByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var user User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	user.ID = id
	if err := s.userService.UpdateUserByID(&user); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func setupRoute(s *Server) *gin.Engine {
	r := gin.Default()
	r.GET("/users", s.AllUsers)
	r.GET("/users/:id", s.GetUserByID)
	r.POST("/users", s.CreateUser)
	r.PUT("/users/:id", s.UpdateUserByID)
	return r
}

func main() {
	host := flag.String("host","localhost","Host")
	port := flag.String("port","8000","Port")
	dbURL := flag.String("dburl","root:root1234@tcp(127.0.0.1:3306)/gotraining","DB Connection")
	flag.Parse()
	addr := fmt.Sprintf("%s:%s", *host, *port)
	db, err := sql.Open("mysql", *dbURL)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}
	s := &Server{
		userService: &UserServiceImp{
			db: db,
		},
		accountService: &AccountServiceImp{
			db: db,
		},
	}
	r := setupRoute(s)
	r.Run(addr)
}
