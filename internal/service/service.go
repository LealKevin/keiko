package service

import (
	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/db"
)

type VocabService interface {
	GetNextWord(levels []int) (data.Word, error)
	MarkWordAsSeen(id int) error
	ResetSeenWords(level int) error
	GetWordsCount() (int, error)
}

type service struct {
	repo *db.DB
}

func New(db *db.DB) VocabService {
	return &service{repo: db}
}

func (s *service) GetNextWord(levels []int) (data.Word, error) {
	word, err := s.repo.GetNextWord(levels)
	if err != nil {
		return data.Word{}, err
	}
	err = s.MarkWordAsSeen(word.ID)
	if err != nil {
		return data.Word{}, err
	}

	return word, nil
}

func (s *service) MarkWordAsSeen(id int) error {
	return s.repo.MarkWordAsSeen(id)
}

func (s *service) ResetSeenWords(level int) error {
	return s.repo.ResetSeenWords(level)
}

func (s *service) GetWordsCount() (int, error) {
	return s.repo.GetWordsCount()
}
