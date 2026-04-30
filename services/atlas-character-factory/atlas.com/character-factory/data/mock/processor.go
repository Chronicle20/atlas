package mock

import (
	"atlas-character-factory/data"
	"context"
)

type FakeClient struct {
	Skills    map[uint32]data.SkillInfo
	Items     map[uint32]data.ItemInfo
	SkillsErr error
}

func (f *FakeClient) GetSkillsByIds(_ context.Context, ids []uint32) ([]data.SkillInfo, error) {
	if f.SkillsErr != nil {
		return nil, f.SkillsErr
	}
	out := make([]data.SkillInfo, 0)
	for _, id := range ids {
		if sk, ok := f.Skills[id]; ok {
			out = append(out, sk)
		}
	}
	return out, nil
}

func (f *FakeClient) GetItemById(_ context.Context, id uint32) (data.ItemInfo, error) {
	if it, ok := f.Items[id]; ok {
		return it, nil
	}
	return data.ItemInfo{}, data.ErrNotFound
}
