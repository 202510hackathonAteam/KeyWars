package handler

import (
  "keywars/backend/internal/service"
)

type API struct {
	Auth *AuthHandler
}

func New(
  authService service.AuthService,
) *API {
  return &API{
    Auth: NewAuthHandler(authService),
  }
}