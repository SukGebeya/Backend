package repository

import (
	"context"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type inventoryRepository struct {
	queries *db.Queries
}

func NewInventoryRepository(pool *pgxpool.Pool) InventoryRepository {
	return &inventoryRepository{queries: db.New(pool)}
}

var _ InventoryRepository = (*inventoryRepository)(nil)

func (r *inventoryRepository) GetByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.Inventory, error) {
	inventory, err := r.queries.GetInventoryByVariantID(ctx, variantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("inventory not found")
		}
		return nil, domainerrors.Internal("failed to get inventory", err)
	}
	return dbInventoryToDomain(inventory), nil
}

func (r *inventoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Inventory, error) {
	inventory, err := r.queries.GetInventoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("inventory not found")
		}
		return nil, domainerrors.Internal("failed to get inventory", err)
	}
	return dbInventoryToDomain(inventory), nil
}

func (r *inventoryRepository) GetByProductID(ctx context.Context, productID uuid.UUID) ([]domain.Inventory, error) {
	inventories, err := r.queries.GetInventoryByProductID(ctx, productID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get inventory by product", err)
	}

	result := make([]domain.Inventory, len(inventories))
	for i, inv := range inventories {
		result[i] = *dbInventoryToDomain(inv)
	}
	return result, nil
}

func (r *inventoryRepository) Create(ctx context.Context, inventory *domain.Inventory) (*domain.Inventory, error) {
	created, err := r.queries.CreateInventory(ctx, db.CreateInventoryParams{
		ID:        inventory.ID,
		VariantID: inventory.VariantID,
		Quantity:  int32(inventory.Quantity),
		Reserved:  int32(inventory.Reserved),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create inventory", err)
	}
	return dbInventoryToDomain(created), nil
}

func (r *inventoryRepository) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.Inventory, error) {
	updated, err := r.queries.UpdateInventoryQuantity(ctx, db.UpdateInventoryQuantityParams{
		ID:       id,
		Quantity: int32(quantity),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.NotFound("inventory not found")
		}
		return nil, domainerrors.Internal("failed to update inventory quantity", err)
	}
	return dbInventoryToDomain(updated), nil
}

func (r *inventoryRepository) Reserve(ctx context.Context, variantID uuid.UUID, qty int) error {
	return r.queries.ReserveInventory(ctx, db.ReserveInventoryParams{
		VariantID: variantID,
		Qty:       int32(qty),
	})
}

func (r *inventoryRepository) Release(ctx context.Context, variantID uuid.UUID, qty int) error {
	return r.queries.ReleaseInventory(ctx, db.ReleaseInventoryParams{
		VariantID: variantID,
		Qty:       int32(qty),
	})
}

func (r *inventoryRepository) Commit(ctx context.Context, variantID uuid.UUID, qty int) error {
	return r.queries.CommitInventory(ctx, db.CommitInventoryParams{
		VariantID: variantID,
		Qty:       int32(qty),
	})
}

func dbInventoryToDomain(inv db.Inventory) *domain.Inventory {
	return &domain.Inventory{
		ID:        inv.ID,
		VariantID: inv.VariantID,
		Quantity:  int(inv.Quantity),
		Reserved:  int(inv.Reserved),
		UpdatedAt: inv.UpdatedAt.Time,
	}
}
