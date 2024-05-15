package resolver

import (
	"asan/graph/db"
	authMiddleware "asan/graph/middleware/auth"
	"asan/graph/model"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/IBM/sarama"
	"github.com/jackc/pgx/v5"
)

var channels = make(map[string]chan *model.Message)

func HandleMessage(ctx context.Context) (<-chan *model.Message, error) {
	userId := "1"
	// userId := middleware.ForContext(ctx)
	// if userId == "" {
	// 	return nil, errors.New("unauthorized 1")
	// }
	// var isInThisRoom bool
	// if err := db.GetPostgresInstance().QueryRow(context.Background(), "select exists (select 1 from user_chat_rooms where id=$1 and user_id=$2)", roomID, userId).Scan(&isInThisRoom); err != nil {
	// 	return nil, errors.New("not in this room")
	// }
	// if !isInThisRoom {
	// 	return nil, errors.New("not in this room")
	// }
	ch := make(chan *model.Message)
	channels[userId] = ch
	go func() {
		defer close(ch)
		defer delete(channels, userId)
		for {
			select {
			case <-ch:
				fmt.Println("123123123")
				// Our message went through
			case <-ctx.Done():
				// Exit on cancellation
				return
			}
		}
		// Wait for context cancellation. Subscription closes.

	}()
	return ch, nil
}

func SendChatMessage(ctx context.Context, message string, roomID string) (bool, error) {
	// userId := authMiddleware.ForContext(ctx)
	// if userId == "" {
	// 	return false, errors.New("unauthorrized")
	// }
	// rows, _ := db.GetPostgresPool().Query(
	// 	context.Background(),
	// 	`SELECT user_id FROM user_chat_rooms WHERE room_id=$1;`,
	// 	roomID)
	// userIds, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
	// 	var userId string
	// 	err := row.Scan(&userId)
	// 	return userId, err
	// })
	// if err != nil {
	// 	return false, err
	// }
	// if !slices.Contains(userIds, userId) {
	// 	return false, errors.New("forbidden")
	// }
	// coll := db.GetMongoClient().Database("asan").Collection("chat")
	// msg := model.SendMessagePayload{Content: message, RoomID: roomID, CreatedAt: time.Now()}
	// if _, err := coll.InsertOne(context.TODO(), msg); err != nil {
	// 	return false, err
	// }
	// for _, id := range userIds {
	// 	if ch := channels[id]; ch != nil && id != userId {
	// 		ch <- &model.Message{Content: message}
	// 	}
	// }

	producer, err := sarama.NewAsyncProducer(strings.Split(os.Getenv("KAFKA_PEERS"), ","), nil)
	if err != nil {
		log.Fatalf("Failed to start Sarama producer: %v", err)
	}

	go func() {
		producer.Input() <- &sarama.ProducerMessage{
			Topic: "chat",
			Value: sarama.StringEncoder(message),
		}
	}()

	return true, nil
}

func CreateChatRoom(ctx context.Context, userIds []string) (bool, error) {
	userId := authMiddleware.ForContext(ctx)
	if userId == "" {
		return false, errors.New("unauthorized")
	}
	tx, err := db.GetPostgresPool().Begin(context.Background())
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())
	var roomId string
	if err := tx.QueryRow(
		context.Background(),
		"insert into chat_rooms default values returning id;",
	).Scan(&roomId); err != nil {
		return false, err
	}
	withThisUserId := append(userIds, userId)
	if _, err := tx.CopyFrom(
		context.Background(),
		pgx.Identifier{"user_chat_rooms"},
		[]string{"room_id", "user_id"},
		pgx.CopyFromSlice(len(withThisUserId), func(i int) ([]any, error) {
			return []any{roomId, withThisUserId[i]}, nil
		})); err != nil {
		return false, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return false, err
	}
	return true, nil
}
