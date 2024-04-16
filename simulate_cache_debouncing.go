package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"log"
	"sync"
	"time"
)

const (
	db1DSN = "host=localhost port=5435 user=user4 password=password4 dbname=mydatabase4 sslmode=disable"
)

var db *sql.DB
var redisClient *redis.Client
var ctx = context.Background()

func initDB() {
	var err error
	db, err = sql.Open("postgres", db1DSN)
	if err != nil {
		log.Fatalf("Could not connect to db1: %v", err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

func insertNewPost(title string, author string) {
	sqlStatement := `INSERT INTO public.posts (title, author) VALUES ($1, $2)`
	_, err := db.Exec(sqlStatement, title, author)
	if err != nil {
		log.Fatalf("Could not insert new post: %v", err)
	}
	log.Printf("New post with title: %s from author: %s inserted successfully", title, author)
}
func fetchPost(clientID int, postID int) (string, error) {

	title, err := redisClient.Get(ctx, fmt.Sprintf("post::%d", postID)).Result()
	if err == nil {
		//log.Printf("found postID: %d in redis", postID)
		return title, nil
	}

	lockKey := fmt.Sprintf("fetching:%d", postID)
	ok, err := redisClient.SetNX(ctx, lockKey, true, 5*time.Second).Result()
	if err != nil {
		log.Printf("Error setting Redis key: %v", err)
		return "", err
	}

	if !ok {
		waitDuration := 100 * time.Millisecond
		timeout := time.After(10 * time.Second)
		for {
			select {
			case <-timeout:
				log.Printf("Timed out waiting for another client to fetch the post id %d", postID)
				return "", fmt.Errorf("timeout waiting for another client")
			default:
				title, err = redisClient.Get(ctx, fmt.Sprintf("post::%d", postID)).Result()
				if err == nil && title != "" {
					return title, nil
				}
				time.Sleep(waitDuration)
			}
		}
	}

	defer redisClient.Del(ctx, lockKey)
	title, err = fetchPostByID(clientID, postID)
	if err != nil {
		return "", err
	}

	redisClient.Set(ctx, fmt.Sprintf("post::%d", postID), title, 30*time.Minute)
	return title, nil
}

func cacheDebounce(postID int) {

	var wg sync.WaitGroup
	startTime := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			title, err := fetchPost(i, postID)
			if err != nil {
				fmt.Printf("Error fetching post for client %d: %v\n", i, err)
			} else {
				fmt.Printf("Client %d fetched title: %s\n", i, title)
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Time for cache debounce completed %d ms\n", time.Since(startTime).Milliseconds())

}

func fetchPostByID(clientID int, postID int) (string, error) {
	log.Printf("fetching post for client ID: %d", clientID)
	sqlStatement := `SELECT title FROM public.posts WHERE id = $1`
	var title string
	err := db.QueryRow(sqlStatement, postID).Scan(&title)
	if err != nil {
		log.Printf("Could not fetch post by post id: %d %v", postID, err)
		return "", err
	}
	return title, nil
}

func fetchPostByTitle(title string) (int, error) {
	sqlStatement := `SELECT id FROM public.posts WHERE title = $1`
	var postID int
	err := db.QueryRow(sqlStatement, title).Scan(&postID)
	if err != nil {
		log.Fatalf("Could not insert new post: %v", err)
		return 0, err
	}
	return postID, nil
}

func main() {
	initDB()
	initRedis()
	insertNewPost("new post from sp", "sp")
	postID, _ := fetchPostByTitle("new post from sp")
	log.Printf("postID %d", postID)
	cacheDebounce(postID)
	redisClient.Del(ctx, fmt.Sprintf("post::%d", postID))
	defer db.Close()
}
