package security

import(
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"golang.org/x/crypto/bcrypt"
	"errors"
	"math/rand"
	"encoding/hex"
)

var ErrNoSuchUser = errors.New("no such user")
var ErrInvalidPassword = errors.New("invalid password")
var ErrTokenExpired = errors.New("token expired")
var ErrInternal = errors.New("internal error")

type Service struct{
	pool *pgxpool.Pool
}

type AuthCredential struct {
	Login 		string		`json:"login"`
	Password 	string		`json:"password"`
}

type TokenData struct {
	Token string `json:"token"`
}


func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Auth(ctx context.Context, login, password string) (bool) {
	count := 0
	
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM managers WHERE login = $1 AND password = $2
		`, login, password).Scan(&count)

	if err != nil {
		log.Print(err)
		return false
	}

	if count == 0 { 
		return false
	}

	return true
}

func (s *Service) TokenForCustomer(ctx context.Context, authCredential *AuthCredential) (token string, err error){
	var hash string
	var id int64
	err = s.pool.QueryRow(ctx, `SELECT id, password FROM customers where phone=$1`, authCredential.Login).Scan(&id, &hash)

	if err == pgx.ErrNoRows {
		return "", ErrNoSuchUser
	}
	if err != nil {
		return "", ErrInternal
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(authCredential.Password))
	if err != nil {
		return "", ErrInvalidPassword
	}

	buffer := make([]byte, 256)
	n, err := rand.Read(buffer)
	if n != len(buffer) || err != nil {
		return "", ErrInternal
	}

	token = hex.EncodeToString(buffer)
	_, err = s.pool.Exec(ctx, `INSERT INTO customers_tokens(token, customer_id) VALUES($1, $2)`, token, id)

	if err != nil {
		return "", ErrInternal
	}

	return token, nil
}

func (s *Service) TokenValidate(ctx context.Context, tokenData *TokenData) (id int64, err error) {
	var diff float32
	err = s.pool.QueryRow(ctx, `SELECT customer_id, EXTRACT(EPOCH FROM (now()-created))/3600 as diff FROM customers_tokens where token=$1`, tokenData.Token).Scan(&id, &diff)
	if err == pgx.ErrNoRows {
		return 0, ErrNoSuchUser
	}

	if diff > 1 {
		return 0, ErrTokenExpired 
	}

	if err != nil {
		return 0, err
	}

	return id, nil
}

