package service

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ InventoryManager = (*InventoryService)(nil)

type InventoryService struct {
	inventoryRepo repository.InventoryRepository
}

func NewInventoryService(inventoryRepo repository.InventoryRepository) *InventoryService {
	return &InventoryService{inventoryRepo: inventoryRepo}
}

func (s *InventoryService) GetByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.Inventory, error) {
	return s.inventoryRepo.GetByVariantID(ctx, variantID)
}

func (s *InventoryService) GetByProductID(ctx context.Context, productID uuid.UUID) ([]domain.Inventory, error) {
	return s.inventoryRepo.GetByProductID(ctx, productID)
}

func (s *InventoryService) Create(ctx context.Context, input CreateInventoryInput) (*domain.Inventory, error) {
	inventory := &domain.Inventory{
		ID:        uuid.New(),
		VariantID: input.VariantID,
		Quantity:  input.Quantity,
		Reserved:  input.Reserved,
	}
	return s.inventoryRepo.Create(ctx, inventory)
}

func (s *InventoryService) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) (*domain.Inventory, error) {
	return s.inventoryRepo.UpdateQuantity(ctx, id, quantity)
}

func (s *InventoryService) Reserve(ctx context.Context, variantID uuid.UUID, qty int) error {
	inv, err := s.inventoryRepo.GetByVariantID(ctx, variantID)
	if err != nil {
		return err
	}
	if !inv.CanReserve(qty) {
		return domainerrors.BadRequest("insufficient available inventory")
	}
	return s.inventoryRepo.Reserve(ctx, variantID, qty)
}

func (s *InventoryService) Release(ctx context.Context, variantID uuid.UUID, qty int) error {
	return s.inventoryRepo.Release(ctx, variantID, qty)
}

func (s *InventoryService) Commit(ctx context.Context, variantID uuid.UUID, qty int) error {
	return s.inventoryRepo.Commit(ctx, variantID, qty)
}
