package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
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
	Content25 string
	Product   *Product
	User      *User
}

type History struct {
	ID        int       `db:"id" json:"id"`
	ProductId int       `db:"product_id" json:"product_id"`
	UserId    int       `db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Product   *Product
	User      *User
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
	Descr70     string
	Comments5   []*Comment
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

var store = sessions.NewCookieStore([]byte(Getenv("SESSION_KEY", "-")))
var sessionName = "session-name"

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
	r.HandleFunc("/products/{id:[0-9]+}", ProductHandler)
	r.HandleFunc("/users/{id:[0-9]+}", UserHandler)
	r.HandleFunc("/login", LoginHandler).Methods("GET")
	r.HandleFunc("/login", LoginPostHandler).Methods("POST")
	r.HandleFunc("/logout", LogoutHandler)
	r.HandleFunc("/products/buy/{id:[0-9]+}", ProductBuyHandler).Methods("POST")
	r.HandleFunc("/comments/{id:[0-9]+}", CommentHandler).Methods("POST")
	r.HandleFunc("/initialize", InitializeHandler)
	r.Use(loggingMiddleware)
	http.Handle("/", r)
	LoadDataCache()
	http.ListenAndServe(":8080", nil)
}

var (
	users          []User
	comments       []Comment
	histories      []History
	products       []Product
	userMap        map[int]*User
	commentMap     map[int]*Comment
	historyMap     map[int]*History
	productMap     map[int]*Product
	userMapByEmail map[string]*User
)

func CutText(text string, length int) string {
	if utf8.RuneCountInString(text) > length {
		return string([]rune(text)[:length]) + "…"
	}
	return text
}

func getCurrentUser(r *http.Request) (user *User) {
	user = &User{}
	session, _ := store.Get(r, sessionName)
	email := session.Values["email"]
	if email != nil {
		u := userMapByEmail[email.(string)]
		if u != nil {
			user = u
		}
	}
	return
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	p := 0
	if page, ok := r.URL.Query()["page"]; ok {
		p, _ = strconv.Atoi(page[0])
	}
	viewProducts := []*Product{}
	for i := 0; i < 50; i++ {
		id := 10000 - p*50 - i
		v := productMap[id]
		viewProducts = append(viewProducts, v)
	}
	data := map[string]interface{}{
		"CurrentUser": currentUser,
		"Products":    viewProducts,
	}
	t := template.Must(template.ParseFiles("templates/layout.tmpl", "templates/index.tmpl"))
	if err := t.Execute(w, data); err != nil {
		log.Println(err)
	}
}

func ProductHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	vars := mux.Vars(r)
	productID, _ := strconv.Atoi(vars["id"])
	product := productMap[productID]
	comments := product.Comments
	bought := false
	data := map[string]interface{}{
		"CurrentUser":   currentUser,
		"Product":       product,
		"Comments":      comments,
		"AlreadyBought": bought,
	}
	t := template.Must(template.ParseFiles("templates/layout.tmpl", "templates/product.tmpl"))
	if err := t.Execute(w, data); err != nil {
		log.Println(err)
	}

}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	vars := mux.Vars(r)
	userID, _ := strconv.Atoi(vars["id"])
	user := userMap[userID]
	totalPay := 0
	histories := []*History{}
	for i := range user.Histories {
		history := user.Histories[len(user.Histories)-i-1]
		histories = append(histories, history)
		totalPay += history.Product.Price
	}
	data := map[string]interface{}{
		"CurrentUser": currentUser,
		"User":        user,
		"Histories":   histories,
		"TotalPay":    totalPay,
	}

	t := template.Must(template.New("layout.tmpl").Funcs(template.FuncMap{
		"jst": func(t time.Time) string {
			return t.Local().Format("2006-01-02 03:04:05")
		},
	}).ParseFiles("templates/layout.tmpl", "templates/mypage.tmpl"))

	if err := t.Execute(w, data); err != nil {
		log.Println(err)
	}

}

func showLoginPage(w http.ResponseWriter, r *http.Request, message string) {
	data := map[string]interface{}{
		"Message": message,
	}
	t := template.Must(template.ParseFiles("templates/login.tmpl"))
	if err := t.Execute(w, data); err != nil {
		log.Println(err)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	session.Values["email"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	showLoginPage(w, r, "ECサイトで爆買いしよう！！！！")
}

func LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")
	log.Println("Auth", email, password)

	session, _ := store.Get(r, sessionName)
	u := userMapByEmail[email]
	if u != nil && u.Password == password {
		session.Values["email"] = email
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		showLoginPage(w, r, "ログインに失敗しました")
	}
}

func ProductBuyHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	if currentUser.ID == 0 {
		showLoginPage(w, r, "先にログインをしてください")
		return
	}
	vars := mux.Vars(r)
	productID, _ := strconv.Atoi(vars["id"])
	log.Printf("Buy %d by %s(%d)", productID, currentUser.Email, currentUser.ID)
	// TODO:同期
	// TODO:DB更新
	user := currentUser
	newHistoryID := histories[len(histories)-1].ID + 1
	product := productMap[productID]
	history := History{ID: newHistoryID, ProductId: productID, UserId: currentUser.ID, Product: product, User: user, CreatedAt: time.Now()}
	histories = append(histories, history)
	historyMap[newHistoryID] = &history
	product.Histories = append(product.Histories, &history)
	user.Histories = append(user.Histories, &history)
	http.Redirect(w, r, "/users/"+strconv.Itoa(currentUser.ID), http.StatusFound)
	log.Printf("Buy success %d by %s(%d)", productID, currentUser.Email, currentUser.ID)
}

func getLast5Comments(product *Product) []*Comment {
	comments5 := []*Comment{}
	for j := 0; j < 5; j++ {
		comments5 = append(comments5, product.Comments[len(product.Comments)-j-1])
	}
	return comments5
}

func CommentHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	if currentUser.ID == 0 {
		showLoginPage(w, r, "先にログインをしてください")
		return
	}
	vars := mux.Vars(r)
	productID, _ := strconv.Atoi(vars["id"])
	r.ParseForm()
	content := r.PostForm.Get("content")
	log.Printf("Post comment %d by %s(%d)", productID, currentUser.Email, currentUser.ID)
	// TODO:同期
	// TODO:DB更新
	user := currentUser
	newCommentID := comments[len(comments)-1].ID + 1
	product := productMap[productID]
	comment := Comment{ID: newCommentID, ProductId: productID, UserId: currentUser.ID, Product: product, User: user, CreatedAt: time.Now(), Content: content, Content25: CutText(content, 25)}
	comments = append(comments, comment)
	commentMap[newCommentID] = &comment
	product.Comments = append(product.Comments, &comment)
	product.Comments5 = getLast5Comments(product)
	user.Comments = append(user.Comments, &comment)
	http.Redirect(w, r, "/users/"+strconv.Itoa(currentUser.ID), http.StatusFound)
	log.Printf("Post comment success %d by %s(%d)", productID, currentUser.Email, currentUser.ID)
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
	userMapByEmail = make(map[string]*User)
	for i := range users {
		v := &users[i]
		userId := v.ID
		v.Comments = make([]*Comment, 0)
		v.Histories = make([]*History, 0)
		userMap[userId] = v
		userMapByEmail[v.Email] = v
	}

	productMap = make(map[int]*Product)
	for i := range products {
		v := &products[i]
		productID := v.ID
		v.Comments = make([]*Comment, 0)
		v.Histories = make([]*History, 0)
		v.Descr70 = CutText(v.Description, 70)
		productMap[productID] = v
	}

	commentMap = make(map[int]*Comment)
	for i := range comments {
		v := &comments[i]
		commentMap[i] = v
		v.Content25 = CutText(v.Content, 25)
		userMap[v.UserId].Comments = append(userMap[v.UserId].Comments, v)
		productMap[v.ProductId].Comments = append(productMap[v.ProductId].Comments, v)
		v.Product = productMap[v.ProductId]
		v.User = userMap[v.UserId]
	}

	historyMap = make(map[int]*History)
	for i := range histories {
		v := &histories[i]
		historyMap[i] = v
		userMap[v.UserId].Histories = append(userMap[v.UserId].Histories, v)
		productMap[v.ProductId].Histories = append(productMap[v.ProductId].Histories, v)
		v.Product = productMap[v.ProductId]
		v.User = userMap[v.UserId]
	}

	for i := range products {
		v := &products[i]
		v.Comments5 = getLast5Comments(v)
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
