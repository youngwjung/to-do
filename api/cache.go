package main

import (
	// "context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	// "time"

	"github.com/gomodule/redigo/redis"
)

type RedisPool interface {
	Get() redis.Conn
}

var ErrCacheMiss = fmt.Errorf("item is not in cache")

func NewCache(redisHost, redisPort string, enabled bool) (*Cache, error) {
	c := &Cache{}
	pool := c.InitPool(redisHost, redisPort)
	c.enabled = enabled
	c.redisPool = pool
	return c, nil
}

type Cache struct {
	redisPool RedisPool
	enabled   bool
}

func (c *Cache) log(msg string) {
	log.Printf("Cache     : %s\n", msg)
}

func (c Cache) InitPool(redisHost, redisPort string) RedisPool {
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	const maxConnections = 10

	pool := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redisAddr)
	}, maxConnections)

	conn := pool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		panic(err)
	}

	msg := fmt.Sprintf("Initialized at %s", redisAddr)
	c.log(msg)
	return pool
}

func (c Cache) Clear() error {
	if !c.enabled {
		return nil
	}
	conn := c.redisPool.Get()
	defer conn.Close()

	if _, err := conn.Do("FLUSHALL"); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Save(todo Todo) error {
	if !c.enabled {
		return nil
	}

	conn := c.redisPool.Get()
	defer conn.Close()

	json, err := todo.JSON()
	if err != nil {
		return fmt.Errorf("cannot convert todo to json: %s", err)
	}

	conn.Send("MULTI")
	conn.Send("SET", strconv.Itoa(todo.ID), json)

	if _, err := conn.Do("EXEC"); err != nil {
		return fmt.Errorf("cannot perform exec operation on cache: %s", err)
	}
	c.log("Successfully saved todo to cache")
	return nil
}

func (c *Cache) Get(key string) (Todo, error) {
	t := Todo{}
	if !c.enabled {
		return t, ErrCacheMiss
	}
	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.String(conn.Do("GET", key))
	if err == redis.ErrNil {
		return Todo{}, ErrCacheMiss
	} else if err != nil {
		return Todo{}, err
	}

	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return Todo{}, err
	}
	c.log("Successfully retrieved todo from cache")

	return t, nil
}

func (c *Cache) Delete(key string) error {
	if !c.enabled {
		return nil
	}
	conn := c.redisPool.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", key); err != nil {
		return err
	}

	c.log(fmt.Sprintf("Cleaning from cache %s", key))
	return nil
}

func (c *Cache) List() (Todos, error) {
	t := Todos{}
	if !c.enabled {
		return t, ErrCacheMiss
	}
	conn := c.redisPool.Get()
	defer conn.Close()

	s, err := redis.String(conn.Do("GET", "todoslist"))
	if err == redis.ErrNil {
		return Todos{}, ErrCacheMiss
	} else if err != nil {
		return Todos{}, err
	}

	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return Todos{}, err
	}
	c.log("Successfully retrieved todos from cache")

	return t, nil
}

func (c *Cache) SaveList(todos Todos) error {
	if !c.enabled {
		return nil
	}

	conn := c.redisPool.Get()
	defer conn.Close()

	json, err := todos.JSON()
	if err != nil {
		return err
	}

	if _, err := conn.Do("SET", "todoslist", json); err != nil {
		return err
	}
	c.log("Successfully saved todo to cache")
	return nil
}

func (c *Cache) DeleteList() error {
	if !c.enabled {
		return nil
	}

	return c.Delete("todoslist")
}
