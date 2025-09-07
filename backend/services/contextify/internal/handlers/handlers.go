package handlers

import (
	"github.com/sujal-lgtm/Contextify/backend/services/contextify/internal/db"
)

type Handlers struct {
	DB *db.DB
}

func NewHandlers(database *db.DB) *Handlers {
	return &Handlers{DB: database}
}
