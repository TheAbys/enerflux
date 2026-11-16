package domainconfig

import (
	"context"
)

type Store interface {
	Get(context.Context) (Settings, error)
	Save(context.Context, Settings) error
}

type Service struct {
	Store Store
}

func NewService(store Store) *Service {
	return &Service{Store: store}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	settings, err := s.Store.Get(ctx)
	if err != nil {
		return Settings{}, err
	}
	if settings.TimeZone == "" {
		settings.TimeZone = "Europe/Berlin"
	}
	return settings, nil
}

func (s *Service) Save(ctx context.Context, settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	return s.Store.Save(ctx, settings)
}
