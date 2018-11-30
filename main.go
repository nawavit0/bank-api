package main

import (
	"encoding/json"
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
	AccountNumber      json.Number    `json:"account_number"`
	Name      string    `json:"name"`
	Balance      int64    `json:"balance"`
}

type Transaction struct {
	Amount	int64			`json:"amount"`
	From	json.Number		`json:"from"`
	To      json.Number		`json:"to"`
}

type UserService interface {
	All() ([]User, error)
	GetUserByID(id int64) (*User, error)
	CreateUser(user *User) error
	UpdateUserByID(user *User) error
	DeleteUserByID(id int64) error
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

func (s *UserServiceImp) GetUserByID(id int64) (*User, error) {
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

func (s *UserServiceImp) DeleteUserByID(id int64) error {
	stmt := "DELETE FROM users WHERE id = ?;"
	_, err := s.db.Exec(stmt, id)
	if err != nil {
		return err
	}
	return nil
}

type AccountService interface {
	CreateAccount(account *BankAccount) error
	GetAccountsByUserID(id int64) ([]BankAccount, error)
	DeleteAccountByUserID(id int64) error
	WithdrawByAccountID(id int64, trans *Transaction) error
	DepositByAccountID(id int64, trans *Transaction) error
	Transfers(trans *Transaction) error
}

type AccountServiceImp struct {
	db *sql.DB
}

func (s *AccountServiceImp) CreateAccount(account *BankAccount) error {
	stmt := "INSERT INTO bank_accounts (user_id, account_number, name, balance) values (?, ?, ?, ?);"
	account_num,_ := account.AccountNumber.Int64()
	res, err := s.db.Exec(stmt, account.UserID, account_num, account.Name, 0)
	if err != nil {
		return err
	}
	account_id, _ := res.LastInsertId()
	account.ID = account_id
	account.Balance = 0
	return nil
}

func (s *AccountServiceImp) GetAccountsByUserID(id int64) ([]BankAccount, error) {
	stmt := "SELECT id, user_id, account_number, name, balance FROM bank_accounts WHERE user_id = ?;"
	rows, err := s.db.Query(stmt, id)
	if err != nil {
		return nil, err
	}
	accounts := []BankAccount{} // set empty slice without nil
	for rows.Next() {
		var account BankAccount
		err := rows.Scan(&account.ID, &account.UserID, &account.AccountNumber, &account.Name, &account.Balance)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (s *AccountServiceImp) DeleteAccountByUserID(id int64) error {
	stmt := "DELETE FROM bank_accounts WHERE user_id = ?;"
	_, err := s.db.Exec(stmt, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *AccountServiceImp) WithdrawByAccountID(id int64, trans *Transaction) error {
	stmt := "UPDATE bank_accounts SET balance = balance - ? WHERE id = ?;"
	_, err := s.db.Exec(stmt, trans.Amount, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *AccountServiceImp) DepositByAccountID(id int64, trans *Transaction) error {
	stmt := "UPDATE bank_accounts SET balance = balance + ? WHERE id = ?;"
	_, err := s.db.Exec(stmt, trans.Amount, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *AccountServiceImp) Transfers(trans *Transaction) error {
	stmtFrom := "UPDATE bank_accounts SET balance = balance - ? WHERE account_number = ?;"
	_, errFrom := s.db.Exec(stmtFrom, trans.Amount, trans.From)
	if errFrom != nil {
		return errFrom
	}
	stmtTo := "UPDATE bank_accounts SET balance = balance + ? WHERE account_number = ?;"
	_, errTo := s.db.Exec(stmtTo, trans.Amount, trans.To)
	if errTo != nil {
		return errTo
	}
	return nil
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
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
	c.JSON(http.StatusOK, user)
}

func (s *Server) DeleteUserByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.userService.DeleteUserByID(id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, id)
}

func (s *Server) CreateAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var account BankAccount
	err := c.ShouldBindJSON(&account)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	account.UserID = id 
	if err := s.accountService.CreateAccount(&account); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, account)
}

func (s *Server) GetAccountsByUserID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	accounts, err := s.accountService.GetAccountsByUserID(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("db: query error: %s", err),
		})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (s *Server) DeleteAccountByUserID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.accountService.DeleteAccountByUserID(id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, id)
}

func (s *Server) WithdrawByAccountID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var trans Transaction
	err := c.ShouldBindJSON(&trans)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	if err := s.accountService.WithdrawByAccountID(id, &trans); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, trans)
}

func (s *Server) DepositByAccountID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var trans Transaction
	err := c.ShouldBindJSON(&trans)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	if err := s.accountService.DepositByAccountID(id, &trans); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, trans)
}

func (s *Server) Transfers(c *gin.Context) {
	var trans Transaction
	err := c.ShouldBindJSON(&trans)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"object":  "error",
			"message": fmt.Sprintf("json: wrong params: %s", err),
		})
		return
	}
	if err := s.accountService.Transfers(&trans); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, trans)
}

func setupRoute(s *Server) *gin.Engine {
	r := gin.Default()
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
        "admin1": "bank",
        "admin2": "bird",
        "admin3": "him",
    }))
	authorized.GET("/users", s.AllUsers)
	authorized.GET("/users/:id", s.GetUserByID)
	authorized.POST("/users", s.CreateUser)
	authorized.PUT("/users/:id", s.UpdateUserByID)
	authorized.DELETE("/users/:id", s.DeleteUserByID)
	authorized.POST("/users/:id/bankAccounts", s.CreateAccount)
	authorized.GET("/users/:id/bankAccounts", s.GetAccountsByUserID)
	authorized.DELETE("/bankAccounts/:id", s.DeleteAccountByUserID)
	authorized.PUT("/bankAccounts/:id/withdraw", s.WithdrawByAccountID)
	authorized.PUT("/bankAccounts/:id/deposit", s.DepositByAccountID)
	authorized.POST("/transfers", s.Transfers)
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
