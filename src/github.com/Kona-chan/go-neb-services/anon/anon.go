// Package anon allows users to send anonymous messages to rooms via bot.
package anon

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/matrix-org/go-neb/types"
	"github.com/matrix-org/gomatrix"
)

const ServiceType = "anon"

type Service struct {
	types.DefaultService
}

const cooldown = time.Duration(1) * time.Minute

var joinedRooms []string
var lastMessage = make(map[string]time.Time)

func (s *Service) Commands(client *gomatrix.Client) []types.Command {
	return []types.Command{
		types.Command{
			Path: []string{"anon", "rooms"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return rooms()
			},
		},
		types.Command{
			Path: []string{"anon"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return parse(client, roomID, userID, args)
			},
		},
	}
}

func (s *Service) Register(oldService types.Service, client *gomatrix.Client) error {
	j, err := client.JoinedRooms()
	if err != nil {
		log.Error(err)
	}
	joinedRooms = j.JoinedRooms
	return nil
}

func parse(client *gomatrix.Client, roomID, userID string, args []string) (interface{}, error) {
	if len(args) < 2 {
		return usage()
	}

	if t, ok := lastMessage[userID]; ok {
		if t.Add(cooldown).After(time.Now()) {
			return failCooldown()
		}
	}
	lastMessage[userID] = time.Now()

	i, err := strconv.Atoi(args[0])
	if !(0 <= i && i < len(joinedRooms)) {
		return rooms()
	}

	destRoomID := joinedRooms[i]
	message := strings.Join(args[1:], " ")
	_, err = client.SendNotice(destRoomID,
		fmt.Sprintf("Просят передать записку следующего содержания:\n%s", message))
	if err != nil {
		return failDelivery(destRoomID, err)
	}

	return success()
}

func rooms() (interface{}, error) {
	return gomatrix.TextMessage{"m.notice",
		fmt.Sprintf("Комнаты, в которые я могу передавать записки:\n%s", joinWithIndex(joinedRooms))}, nil
}

func usage() (interface{}, error) {
	return gomatrix.TextMessage{"m.notice", "Доступные команды:\n!anon rooms\n!anon room_index message"}, nil
}

func failCooldown() (interface{}, error) {
	return gomatrix.TextMessage{"m.notice", "Слишком много записок, умерьте пыл."}, nil
}

func failDelivery(roomID string, err error) (interface{}, error) {
	return gomatrix.TextMessage{"m.notice",
		fmt.Sprintf("Не удалось доставить записку в %s: %s", roomID, err.Error())}, nil
}

func success() (interface{}, error) {
	return gomatrix.TextMessage{"m.notice", "Принято."}, nil
}

func joinWithIndex(ss []string) string {
	var result []string
	for i, s := range ss {
		result = append(result, fmt.Sprintf("%d. %s", i, s))
	}
	return strings.Join(result, "\n")
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &Service{
			DefaultService: types.NewDefaultService(serviceID, serviceUserID, ServiceType),
		}
	})
}
