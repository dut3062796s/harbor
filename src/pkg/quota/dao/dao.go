// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"context"
	"time"

	"github.com/goharbor/harbor/src/internal/orm"
	"github.com/goharbor/harbor/src/pkg/quota/models"
	"github.com/goharbor/harbor/src/pkg/types"
)

// DAO the dao for Quota and QuotaUsage
type DAO interface {
	// Create create quota for reference object
	Create(ctx context.Context, reference, referenceID string, hardLimits, used types.ResourceList) (int64, error)

	// Delete delete quota by id
	Delete(ctx context.Context, id int64) error

	// Get returns quota by id
	Get(ctx context.Context, id int64) (*models.Quota, error)

	// GetForUpdate get quota by reference object and lock it for update
	GetForUpdate(ctx context.Context, reference, referenceID string) (*models.Quota, error)

	// Update update quota
	Update(ctx context.Context, quota *models.Quota) error
}

// New returns an instance of the default DAO
func New() DAO {
	return &dao{}
}

type dao struct{}

func (d *dao) Create(ctx context.Context, reference, referenceID string, hardLimits, used types.ResourceList) (int64, error) {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return 0, err
	}

	now := time.Now()

	quota := &Quota{
		Reference:    reference,
		ReferenceID:  referenceID,
		Hard:         hardLimits.String(),
		CreationTime: now,
		UpdateTime:   now,
	}

	id, err := o.Insert(quota)
	if err != nil {
		return 0, err
	}

	usage := &QuotaUsage{
		ID:           id,
		Reference:    reference,
		ReferenceID:  referenceID,
		Used:         used.String(),
		CreationTime: now,
		UpdateTime:   now,
	}

	_, err = o.Insert(usage)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (d *dao) Delete(ctx context.Context, id int64) error {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return err
	}

	quota := &Quota{ID: id}
	if _, err := o.Delete(quota, "id"); err != nil {
		return err
	}

	usage := &QuotaUsage{ID: id}
	if _, err := o.Delete(usage, "id"); err != nil {
		return err
	}

	return nil
}

func (d *dao) Get(ctx context.Context, id int64) (*models.Quota, error) {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	quota := &Quota{ID: id}
	if err := o.Read(quota); err != nil {
		return nil, orm.WrapNotFoundError(err, "quota %d not found", id)
	}

	usage := &QuotaUsage{ID: id}
	if err := o.Read(usage); err != nil {
		return nil, orm.WrapNotFoundError(err, "quota usage %d not found", id)
	}

	return toQuota(quota, usage), nil
}

func (d *dao) GetForUpdate(ctx context.Context, reference, referenceID string) (*models.Quota, error) {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	quota := &Quota{Reference: reference, ReferenceID: referenceID}
	if err := o.ReadForUpdate(quota, "reference", "reference_id"); err != nil {
		return nil, orm.WrapNotFoundError(err, "quota not found for (%s, %s)", reference, referenceID)
	}

	usage := &QuotaUsage{Reference: reference, ReferenceID: referenceID}
	if err := o.ReadForUpdate(usage, "reference", "reference_id"); err != nil {
		return nil, orm.WrapNotFoundError(err, "quota usage not found for (%s, %s)", reference, referenceID)
	}

	return toQuota(quota, usage), nil
}

func (d *dao) Update(ctx context.Context, quota *models.Quota) error {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return err
	}

	if quota.UsedChanged {
		usage := &QuotaUsage{ID: quota.ID, Used: quota.Used, UpdateTime: quota.UpdateTime}

		_, err := o.Update(usage, "used", "update_time")
		if err != nil {
			return err
		}
	}

	if quota.HardChanged {
		md := &Quota{ID: quota.ID, Hard: quota.Hard, UpdateTime: quota.UpdateTime}

		_, err := o.Update(md, "hard", "update_time")
		if err != nil {
			return err
		}
	}

	return nil
}

func toQuota(quota *Quota, usage *QuotaUsage) *models.Quota {
	return &models.Quota{
		ID:           quota.ID,
		Reference:    quota.Reference,
		ReferenceID:  quota.ReferenceID,
		Hard:         quota.Hard,
		Used:         usage.Used,
		CreationTime: quota.CreationTime,
	}
}