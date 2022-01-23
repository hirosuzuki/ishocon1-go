package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type User struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"password"`
	LastLogin time.Time `db:"last_login" json:"last_login"`
	Comments  []*Comment
	Histories []*History
}

type Comment struct {
	ID        int       `db:"id" json:"id"`
	ProductId int       `db:"product_id" json:"product_id"`
	UserId    int       `db:"user_id" json:"user_id"`
	Content   string    `db:"content" json:"content"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type History struct {
	ID        int       `db:"id" json:"id"`
	ProductId int       `db:"product_id" json:"product_id"`
	UserId    int       `db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Product struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	ImagePath   string    `db:"image_path" json:"image_path"`
	Price       int       `db:"price" json:"price"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	Comments    []*Comment
	Histories   []*History
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func Getenv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	} else {
		return val
	}
}

var db *sqlx.DB

func init() {
	host := Getenv("DB_HOST", "127.0.0.1")
	port := Getenv("DB_PORT", "3306")
	user := Getenv("DB_USER", "ishocon")
	pass := Getenv("DB_PASS", "ishocon")
	name := Getenv("DB_NAME", "ishocon1")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, name)

	var err error
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(10 * time.Second)
}

func main() {
	log.Println("webapp")
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/initialize", InitializeHandler)
	r.Use(loggingMiddleware)
	http.Handle("/", r)
	LoadDataCache()
	http.ListenAndServe(":8080", nil)
}

var (
	users      []User
	comments   []Comment
	histories  []History
	products   []Product
	userMap    map[int]*User
	commentMap map[int]*Comment
	historyMap map[int]*History
	productMap map[int]*Product
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	layout := "templates/layout.tmpl"
	data := map[string]interface{}{
		"CurrentUser": userMap[1],
		"Products":    "interface{}",
	}
	t := template.Must(template.ParseFiles(layout, "templates/index.tmpl"))
	if err := t.Execute(w, data); err != nil {
		log.Println(err)
	}
	w.WriteHeader(200)
	log.Println("////", t)
}

func LoadDataCache() {
	log.Println("Load data cache")

	users = []User{}
	comments = []Comment{}
	histories = []History{}
	products = []Product{}

	if err := db.Select(&users, "SELECT id, name, email, password, last_login FROM users ORDER BY id"); err != nil {
		log.Panicln(err)
	}

	if err := db.Select(&comments, "SELECT id, product_id, user_id, content, created_at FROM comments ORDER BY id"); err != nil {
		log.Panicln(err)
	}

	if err := db.Select(&histories, "SELECT id, product_id, user_id, created_at FROM histories ORDER BY id"); err != nil {
		log.Panicln(err)
	}

	if err := db.Select(&products, "SELECT id, name, description, image_path, price, created_at FROM products ORDER BY id"); err != nil {
		log.Panicln(err)
	}

	userMap = make(map[int]*User)
	for _, v := range users {
		userId := v.ID
		userMap[userId] = &v
		userMap[userId].Comments = make([]*Comment, 0)
		userMap[userId].Histories = make([]*History, 0)
	}

	productMap = make(map[int]*Product)
	for _, v := range products {
		productID := v.ID
		productMap[productID] = &v
		productMap[productID].Comments = make([]*Comment, 0)
		productMap[productID].Histories = make([]*History, 0)
	}

	commentMap = make(map[int]*Comment)
	for i, v := range comments {
		commentMap[i] = &v
		userMap[v.UserId].Comments = append(userMap[v.UserId].Comments, &v)
		productMap[v.ProductId].Comments = append(productMap[v.ProductId].Comments, &v)
	}

	historyMap = make(map[int]*History)
	for i, v := range histories {
		historyMap[i] = &v
		userMap[v.UserId].Histories = append(userMap[v.UserId].Histories, &v)
		productMap[v.ProductId].Histories = append(productMap[v.ProductId].Histories, &v)
	}

	log.Printf("Loaded data cache: users=%d commetns=%d histories=%d products=%d\n", len(users), len(comments), len(histories), len(products))
}

func InitializeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Initialize DB")
	db.MustExec("DELETE FROM users WHERE id > 5000")
	db.MustExec("DELETE FROM products WHERE id > 10000")
	db.MustExec("DELETE FROM comments WHERE id > 200000")
	db.MustExec("DELETE FROM histories WHERE id > 500000")
	LoadDataCache()
}
