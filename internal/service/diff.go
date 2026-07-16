package service

import (
	"cmp"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"slices"
)

func IndexFiles(files []model.FileEntry) map[string]model.FileEntry {
	res := make(map[string]model.FileEntry)

	for _, file := range files {
		res[file.Path] = file
	}

	return res
}

func DiffSnapshots(oldSnap, newSnap model.Snapshot) []model.FileChange {
	res := make([]model.FileChange, 0)

	oldIndex := IndexFiles(oldSnap.Files)
	newIndex := IndexFiles(newSnap.Files)

	for _, file := range newSnap.Files {
		v, ok := oldIndex[file.Path]
		if !ok {
			res = append(res, model.FileChange{
				Path:    file.Path,
				Status:  model.StatusAdded,
				OldHash: "",
				NewHash: file.Hash,
			})
		} else {
			if v.Hash != file.Hash {
				res = append(res, model.FileChange{
					Path:    file.Path,
					Status:  model.StatusModified,
					OldHash: v.Hash,
					NewHash: file.Hash,
				})
			}
		}
	}

	for _, file := range oldSnap.Files {
		_, ok := newIndex[file.Path]
		if !ok {
			res = append(res, model.FileChange{
				Path:    file.Path,
				Status:  model.StatusDeleted,
				OldHash: file.Hash,
				NewHash: "",
			})
		}
	}

	slices.SortFunc(res, func(a, b model.FileChange) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return res
}
