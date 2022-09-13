package revtr_test

import (
	"context"
	"database/sql"
	"fmt"

	revtrpb "github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
)

func InsertTestUser(db *sql.DB, user revtrpb.RevtrUser) int64 {

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	// Create a fake traceroute and fake RR pings
	insertUserQuery := fmt.Sprintf("INSERT INTO users (name, `key`, max_revtr_per_day, revtr_run_today, max_parallel_revtr) VALUES (\"%s\",\"%s\",%d, %d, %d)",
	user.Name, user.Key, user.MaxRevtrPerDay, user.RevtrRunToday, user.MaxParallelRevtr)
	
	res, err := tx.Exec(insertUserQuery)
	fmt.Printf("%s", insertUserQuery)
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		panic(err)
	}


	err =  tx.Commit()
	if err != nil {
		panic(err)
	}

	return id 
}

func DeleteTestUser(db *sql.DB, userID int64) {
	
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	deleteQuery := fmt.Sprintf("DELETE FROM users where id=%d", userID)

	_, err = tx.Exec(deleteQuery)
	fmt.Printf("%s", deleteQuery)
	if err != nil {
		tx.Rollback()
		panic(err)
	}

	err =  tx.Commit()
	if err != nil {
		panic(err)
	}

}