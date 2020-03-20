package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/honeycombio/beeline-go/wrappers/hnysqlx"
	"github.com/jmoiron/sqlx"
)

func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	beeline.AddField(r.Context(), "hello", rand.Intn(1000))
	fmt.Fprintln(w, "Hello World")
}
func AddCustomerHandler(w http.ResponseWriter, r *http.Request) {

	odb, err := sqlx.Connect("mysql", "root:abc123@(localhost:3306)/foo")
	if err != nil {
		fmt.Println(err)
	}
	db := hnysqlx.WrapDB(odb)
	ctx, span := beeline.StartSpan(context.Background(), "start")
	defer span.Send()
	query := `INSERT INTO customers(name,address) VALUES (?,?)`
	db.MustExecContext(ctx, query, "Random Name", "123 4th St Los Angeles CA 90505")
	db.Close()

}

func GetCustomersHandler(w http.ResponseWriter, r *http.Request) {

	type Customers struct {
		name    string
		address string
	}

	ctx, span := beeline.StartSpan(context.Background(), "GetCustomersQuery")
	defer span.Send()
	odb, err := sqlx.Connect("mysql", "root:abc123@(localhost:3306)/foo")
	if err != nil {
		fmt.Println(err)
	}
	db := hnysqlx.WrapDB(odb)
	rows, _ := db.QueryxContext(ctx, "SELECT * FROM customers")
	for rows.Next() {
		var c Customers
		err = rows.StructScan(&c)
		fmt.Println(c.name, c.address)
	}

}
func main() {
	beeline.Init(beeline.Config{
		WriteKey: "b8956d3b0b1b47e470515d03dc1a2330",
		Dataset:  "HelloWorldApp",
	})
	defer beeline.Close()
	r := mux.NewRouter()
	r.HandleFunc("/hello", HelloWorldHandler)
	r.HandleFunc("/create", AddCustomerHandler)
	r.HandleFunc("/read", GetCustomersHandler)
	fmt.Println("Starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(r)))
}
