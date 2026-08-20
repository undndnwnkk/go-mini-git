package service

import (
	"context"
	"github.com/undndnwnkk/go-mini-git/internal/config"
	"github.com/undndnwnkk/go-mini-git/internal/model"
)

type VCSService struct {
	cfg *config.Config
}

type SnapshotOptions struct {
	Workers int
}

func NewVCSService(cfg *config.Config) *VCSService {
	return &VCSService{cfg: cfg}
}

func (s *VCSService) ListSnapshots() ([]model.Snapshot, error) {
	return ListSnapshots(s.cfg.SnapshotsDir())
}

func (s *VCSService) GetSnapshotByID(id string) (model.Snapshot, error) {
	return LoadSnapshotByID(s.cfg.SnapshotsDir(), id)
}

func (s *VCSService) DiffSnapshotsByID(oldID, newID string) ([]model.FileChange, error) {
	oldSnap, err := s.GetSnapshotByID(oldID)
	if err != nil {
		return nil, err
	}

	newSnap, err := s.GetSnapshotByID(newID)
	if err != nil {
		return nil, err
	}

	return DiffSnapshots(oldSnap, newSnap), nil
}

func (s *VCSService) RestoreSnapshotByID(ctx context.Context, snapshotID, targetDir string) error {
	snap, err := s.GetSnapshotByID(snapshotID)
	if err != nil {
		return err
	}

	return RestoreSnapshotWithContext(ctx, snap, targetDir, s.cfg.ObjectsDir())
}

func (s *VCSService) CreateSnapshot(ctx context.Context, root string, opts SnapshotOptions) (model.Snapshot, error) {
	data, err := BuildSnapshotWithContextAndOptions(ctx, root, CollectOptions{Workers: opts.Workers})
	if err != nil {
		return model.Snapshot{}, err
	}

	if err := SaveObjectsWithContext(ctx, root, data.Files, s.cfg.ObjectsDir()); err != nil {
		return model.Snapshot{}, err
	}

	if err := SaveSnapshot(data, s.cfg.SnapshotsDir()); err != nil {
		return model.Snapshot{}, err
	}

	return data, nil
}
