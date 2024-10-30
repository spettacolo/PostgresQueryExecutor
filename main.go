// Change the PostgreSQL port in the connection string if you are using a different port. (lines 45 & 154)

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var clear map[string]func()

func init() {
	clear = make(map[string]func())
	clear["linux"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func callClear() {
	value, ok := clear[runtime.GOOS]
	if ok {
		value()
	} else {
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

func listDatabases() {
	conn := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	var db *sql.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", conn)
		if err != nil {
			log.Printf("Database connection error: %v", err)
			return
		}
		defer db.Close()

		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("Ping error to database: %v. Retry in 2 seconds...", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Printf("Ping database error after 5 attempts: %v", err)
		return
	}

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		log.Printf("Error obtaining database list: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("Available databases:")
	for rows.Next() {
		var datname string
		if err := rows.Scan(&datname); err != nil {
			log.Printf("Error reading database name: %v", err)
		}
		fmt.Println("-", datname)
	}
}

func executeQuery(db *sql.DB, query string) {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Query execution error: %v", err)
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			log.Printf("Column reading error: %v", err)
			return
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		fmt.Println(strings.Join(columns, "\t"))
		fmt.Println(strings.Repeat("-", 80))

		for rows.Next() {
			err := rows.Scan(valuePtrs...)
			if err != nil {
				log.Printf("Error reading line: %v", err)
				continue
			}

			var rowValues []string
			for _, val := range values {
				rowValues = append(rowValues, fmt.Sprintf("%v", val))
			}
			fmt.Println(strings.Join(rowValues, "\t"))
		}
	} else {
		result, err := db.Exec(query)
		if err != nil {
			log.Printf("Query execution error: %v", err)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Query executed successfully. Affected lines: %d\n", rowsAffected)
	}
}

func main() {
	startPostgres()

	var db *sql.DB
	var err error
	var dbname string

	for {
		listDatabases()

		fmt.Print("Enter the name of the database to connect to (or leave blank to connect to 'postgres'): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		dbname = scanner.Text()
		if dbname == "" {
			dbname = "postgres"
		}

		conn := fmt.Sprintf("user=postgres password=postgres dbname=%s host=127.0.0.1 port=5432 sslmode=disable", dbname)
		db, err = sql.Open("postgres", conn)
		if err != nil {
			log.Printf("Database connection error: %v", err)
			continue
		}

		err = db.Ping()
		if err != nil {
			log.Printf("Ping error to database: %v", err)
			continue
		}

		fmt.Printf("Connected to the database '%s'\n", dbname)
		break
	}
	defer db.Close()

	for {
		fmt.Println("\nType your query (end with $send on a new line to execute, $exit to exit, $change_db to change database):")
		var queryLines []string
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			line := scanner.Text()

			if line == "$send" {
				query := strings.Join(queryLines, " ")
				if query != "" {
					executeQuery(db, query)
				}
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()
				break
			} else if line == "$exit" {
				return
			} else if line == "$change_db" {
				db.Close()
				main()
				return
			}

			queryLines = append(queryLines, line)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Input reading error: %v", err)
		}

		callClear()
	}
}

func startPostgres() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("C:\\Program Files\\PostgreSQL\\17\\bin\\pg_ctl", "status", "-D", "C:\\Program Files\\PostgreSQL\\17\\data")
	} else {
		cmd = exec.Command("/usr/local/pgsql/bin/pg_ctl", "status", "-D", "/usr/local/pgsql/data")
	}
	err := cmd.Run()
	if err == nil {
		fmt.Println("PostgreSQL server is already running.")
		return
	} else {
		fmt.Println("Starting PostgreSQL server...")
	}

	if runtime.GOOS == "windows" {
		cmd = exec.Command("C:\\Program Files\\PostgreSQL\\17\\bin\\pg_ctl", "start", "-D", "C:\\Program Files\\PostgreSQL\\17\\data")
	} else {
		cmd = exec.Command("/usr/local/pgsql/bin/pg_ctl", "start", "-D", "/usr/local/pgsql/data")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to start PostgreSQL server: %v", err)
	}
}
