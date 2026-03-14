package add

import (
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/service"
)

func taskErrorText(err error) string {
	if errors.Is(err, service.ErrTaskAlreadyExists) {
		return "Данная задача уже добавлена. Используй /update"
	} else if errors.Is(err, service.ErrFailedUserCreate) {
		return "Не смог сохранить пользователя 😔 Попробуй позже."
	} else if errors.Is(err, service.ErrTgUserNotFound) {
		return "Пользователь не найден 😔"
	} else {
		return "Ошибка 😔 Попробуй позже."
	}
}
