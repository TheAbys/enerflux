package sourceconfig

import (
	"context"
	"fmt"
)

type Service struct {
	Repository *Repository
	Fallback   Settings
}

func NewService(repository *Repository, fallback Settings) *Service {
	return &Service{Repository: repository, Fallback: fallback}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	settings, err := s.Repository.Get(ctx, s.Fallback)
	if err != nil {
		return Settings{}, err
	}
	if err := settings.NormalizeForRead(); err != nil {
		return Settings{}, fmt.Errorf("stored source configuration is invalid: %w", err)
	}
	return settings, nil
}

func (s *Service) Save(ctx context.Context, settings Settings) error {
	if err := settings.NormalizeAndValidate(); err != nil {
		return err
	}
	return s.Repository.Save(ctx, settings)
}
