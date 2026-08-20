package service

import (
	"context"
	"github.com/undndnwnkk/go-mini-git/internal/config"
	"github.com/undndnwnkk/go-mini-git/internal/model"
)

type VCSService struct {
	cfg *config.Config
}

func NewVCSService(cfg *config.Config) *VCSService {
	return &VCSService{cfg: cfg}
}

func (s *VCSService) ListSnapshots() ([]model.Snapshot, error) {
	return ListSnapshots(s.cfg.SnapshotsDir())
}

func (s *VCSService) CreateSnapshot(ctx context.Context, root string) error {
	data, err := BuildSnapshotWithContext(ctx, root)
	if err != nil {
		return err
	}

	// Сохраняем, используя пути из конфига
	if err := SaveObjects(root, data.Files, s.cfg.ObjectsDir()); err != nil {
		return err
	}
	return SaveSnapshot(data, s.cfg.SnapshotsDir())
}
