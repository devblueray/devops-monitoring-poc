package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/honeycombio/beeline-go/wrappers/hnysqlx"
	"github.com/jmoiron/sqlx"
)

func HelloWorldHelper(ctx context.Context) string {
	ctx, span := beeline.StartSpan(ctx, "HelloHelper")
	defer span.Send()
	return "Hello World"
}
func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, HelloWorldHelper(r.Context()))
}

func ThirdPartyAPIHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(r.Context(), "staging ping")
	defer span.Send()
	resp, err := http.Get("http://staging.droplive.com/api/v1/pingall")
	if err != nil {
		beeline.AddField(ctx, "error", err)
		fmt.Println(err)
	}
	apiResponse, _ := json.Marshal(resp)
	w.Write(apiResponse)
}
func AddCustomerHandler(w http.ResponseWriter, r *http.Request) {
	// ctx, span := beeline.StartSpan(context.Background(), "start")
	// defer span.Send()
	type Customer struct {
		Name    string
		Address string
	}
	var c Customer
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		fmt.Println(err)
	}
	odb, err := sqlx.Connect("mysql", "root:abc123@(localhost:3306)/foo")
	if err != nil {
		beeline.AddFieldToTrace(r.Context(), "error", err)
		fmt.Println(err)
	}
	db := hnysqlx.WrapDB(odb)
	defer db.Close()

	query := `INSERT INTO customers(name,address) VALUES (?,?)`
	db.MustExecContext(r.Context(), query, c.Name, c.Address)
}
func Uinttest(w http.ResponseWriter, r *http.Request) {
	ctx, span := beeline.StartSpan(context.Background(), "Uint64Test")
	defer span.Send()
	odb, err := sqlx.Connect("mysql", "root:abc123@(localhost:3306)/foo")
	if err != nil {
		beeline.AddFieldToTrace(ctx, "error", err)
		fmt.Println(err)
	}
	db := hnysqlx.WrapDB(odb)
	defer db.Close()
	var result uint64
	err = db.GetContext(ctx, &result, "SELECT ?", ^uint64(0))
	if err != nil {
		beeline.AddFieldToTrace(ctx, "error", err)
		fmt.Println(err)
	}
	fmt.Println(result)
}
func GetCustomersHandler(w http.ResponseWriter, r *http.Request) {

	type Customers struct {
		ID      string
		Name    string
		Address string
	}
	c := []Customers{}

	ctxQuery, span := beeline.StartSpan(r.Context(), "GetCustomersQuery")
	defer span.Send()
	ctxPing, span := beeline.StartSpan(ctxQuery, "staging ping")
	defer span.Send()
	_, err := http.Get("http://staging.droplive.com/api/v1/pingall")
	if err != nil {
		beeline.AddField(ctxPing, "error", err)
	}
	odb, err := sqlx.Connect("mysql", "root:abc123@(localhost:3306)/foo")
	if err != nil {
		fmt.Println(err)
		beeline.AddFieldToTrace(r.Context(), "error", err)
	}
	db := hnysqlx.WrapDB(odb)
	//defer db.Close()
	err = db.SelectContext(ctxQuery, &c, "SELECT * FROM customers where name = 'nsuoziwfzq'")
	if err != nil {
		beeline.AddFieldToTrace(ctxQuery, "error", err)
		fmt.Println(err)
	}
	customerJSON, err := json.Marshal(c)
	if err != nil {
		beeline.AddFieldToTrace(r.Context(), "error", err)
		fmt.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(customerJSON)

	for _, customer := range c {
		fmt.Println(customer.Name)
	}
}
func main() {
	beeline.Init(beeline.Config{
		WriteKey: "b8956d3b0b1b47e470515d03dc1a2330",
		Dataset:  "HelloWorldApp",
	})
	defer beeline.Close()
	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	r.HandleFunc("/hello", HelloWorldHandler)
	r.HandleFunc("/create", AddCustomerHandler)
	r.HandleFunc("/read", GetCustomersHandler)
	r.HandleFunc("/test", Uinttest)
	r.HandleFunc("/apitest", ThirdPartyAPIHandler)
	fmt.Println("Starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(r)))
}
