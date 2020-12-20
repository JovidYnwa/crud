package customers

import(
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"time"
	"log"
)

var ErrNotFound = errors.New("item not found")
var ErrInternal = errors.New("iternal error")

type Service struct{
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Customer struct {
	ID 		int64 		`json:"id"`
	Name	string 		`json:"name"`
	Phone 	string		`json:"phone"`
	Password string		`json:"password"`
	Active 	bool		`json:"active"`
	Created	time.Time	`json:"created"`
}

func (s *Service) ByID(ctx context.Context, id int64) (*Customer, error) {
	item := &Customer{}
	
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, phone, active, created FROM customers WHERE id = $1
		`, id).Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)

		if errors.Is(err, pgx.ErrNoRows){
			return nil, ErrNotFound
		}

		if err != nil {
			log.Print(err)
			return nil, ErrInternal
		}

		return item, nil
}

//All возращает все клиенты
func (s *Service) All(ctx context.Context) ([]*Customer, error) {
	items := make([]*Customer, 0)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, phone, active, created FROM customers
	`)

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	
	defer rows.Close()


	for rows.Next(){
		item := &Customer{}
		err = rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)
		if err != nil {
			log.Print(err)
			return nil, ErrInternal
		}
		items = append(items, item)
	}

	err = rows.Err()
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	return items, nil
}

//AllActive возращает все активные клиенты
func (s *Service) AllActive(ctx context.Context) ([]*Customer, error) {
	items := make([]*Customer, 0)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, phone, active, created FROM customers WHERE active = true
	`)

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	defer rows.Close()

	for rows.Next(){
		item := &Customer{}
		err = rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)
		if err != nil {
			log.Print(err)
			return nil, ErrInternal
		}
		items = append(items, item)
	}

	err = rows.Err()
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	return items, nil
}

//Save сохраняет/обновляет клиента
func (s *Service) Save(ctx context.Context, item *Customer) (*Customer, error) {
	count := 0
	lastInsertId := 0
	
	
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM customers WHERE id = $1
		`, item.ID).Scan(&count)

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	

	if(count == 0){
		hash, err := bcrypt.GenerateFromPassword([]byte(item.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Print(err)
			return nil, ErrInternal
		}

		err = s.pool.QueryRow(ctx, `INSERT INTO customers(name, phone, password) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING id`, item.Name, item.Phone, hash).Scan(&lastInsertId)

		if err != nil {
			log.Print(err)
			return nil, ErrInternal
		}
		
		newCustomer, err := s.ByID(ctx, int64(lastInsertId))
		if err != nil {
			log.Print(err)
			return nil, err
		}
		
		return newCustomer, nil
	}

	_, err = s.pool.Exec(ctx, `
			UPDATE customers SET name = $1, phone = $2 WHERE id = $3
	`, item.Name, item.Phone, item.ID)

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	updatedCustomer, err := s.ByID(ctx, int64(item.ID))
	if err != nil {
		log.Print(err)
		return nil, err
	}

	return updatedCustomer, nil
}

//RemoveByID удаляет клиент по идентификатору
func (s *Service) RemoveByID(ctx context.Context, id int64) (*Customer, error) {
	customer, err := s.ByID(ctx, id)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	_, err = s.pool.Exec(ctx, `
			DELETE FROM customers WHERE id = $1
	`, id)

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	return customer, nil 
}

func (s *Service) SetStatus(ctx context.Context, id int64, status bool) (*Customer, error) {
	item := &Customer{}

	err := s.pool.QueryRow(ctx, `UPDATE customers SET active=$1 WHERE id=$2 RETURNING id, name, phone, active, created`, status, id).Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)

	if errors.Is(err, pgx.ErrNoRows){
		return nil, ErrNotFound
	}

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	return item, nil 
}