package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
)

type SQLStorage struct {
	db *sql.DB
}

func (s *SQLStorage) log(msg string) {
	log.Printf("MySQL     : %s\n", msg)
}

func (s *SQLStorage) Init(dbUser, dbPassword, dbHost, dbPort, dbName string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	pool, err := sql.Open("mysql", dsn)

	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	if err := pool.PingContext(ctx); err != nil {
		panic(err)
	}

	s.db = pool

	msg := fmt.Sprintf("Initialized at %s:%s", dbHost, dbPort)
	s.log(msg)

	return nil
}

func (s *SQLStorage) Close() error {
	return s.db.Close()
}

func (s SQLStorage) List() (Todos, error) {
	ts := Todos{}
	results, err := s.db.Query("SELECT * FROM todo ORDER BY updated DESC")
	if err != nil {
		return ts, err
	}

	for results.Next() {
		t, err := resultToTodo(results)
		if err != nil {
			return ts, err
		}

		ts = append(ts, t)
	}
	return ts, nil
}

func (s SQLStorage) Create(t Todo) (Todo, error) {
	sql := `
		INSERT INTO todo(title, updated) 
		VALUES(?,NOW())	
	`

	if t.Complete {
		sql = `
		INSERT INTO todo(title, updated, completed) 
		VALUES(?,NOW(),NOW())	
	`
	}

	op, err := s.db.Prepare(sql)
	if err != nil {
		return t, err
	}

	results, err := op.Exec(t.Title)

	id, err := results.LastInsertId()
	if err != nil {
		return t, err
	}

	t.ID = int(id)

	return t, nil
}

func resultToTodo(results *sql.Rows) (Todo, error) {
	t := Todo{}
	if err := results.Scan(&t.ID, &t.Title, &t.Updated, &t.completedNull); err != nil {
		return t, err
	}

	if t.completedNull.Valid {
		t.Completed = t.completedNull.Time
		t.Complete = true
	}

	return t, nil
}

func (s SQLStorage) Read(id string) (Todo, error) {
	t := Todo{}
	results, err := s.db.Query("SELECT * FROM todo WHERE id =?", id)
	if err != nil {
		return t, err
	}

	results.Next()
	t, err = resultToTodo(results)
	if err != nil {
		return t, err
	}

	return t, nil
}

func (s SQLStorage) Update(t Todo) error {
	orig, err := s.Read(strconv.Itoa(t.ID))
	if err != nil {
		return err
	}

	sql := `
		UPDATE todo
		SET title = ?, updated = NOW() 
		WHERE id = ?
	`

	if t.Complete && !orig.Complete {
		sql = `
		UPDATE todo
		SET title = ?, updated = NOW(), completed = NOW() 
		WHERE id = ?
	`
	}

	if orig.Complete && !t.Complete {
		sql = `
		UPDATE todo
		SET title = ?, updated = NOW(), completed = NULL 
		WHERE id = ?
	`
	}

	op, err := s.db.Prepare(sql)
	if err != nil {
		return err
	}

	_, err = op.Exec(t.Title, t.ID)

	if err != nil {
		return err
	}

	return nil
}

func (s SQLStorage) Delete(id string) error {
	op, err := s.db.Prepare("DELETE FROM todo WHERE id =?")
	if err != nil {
		return err
	}

	if _, err = op.Exec(id); err != nil {
		return err
	}

	return nil
}
