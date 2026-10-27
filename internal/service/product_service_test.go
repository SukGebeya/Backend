package service

import (
	"testing"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/stretchr/testify/require"
)

func TestProductDelete_RemovesUnorderedProduct(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 50000, 5)
	svc := NewProductService(h.products, h.variants, nil)

	require.NoError(t, svc.Delete(h.ctx, f.product.ID))

	_, err := h.products.GetByID(h.ctx, f.product.ID)
	require.True(t, domainerrors.IsNotFound(err), "got %v", err)
	_, err = h.variants.GetByID(h.ctx, f.variant.ID)
	require.True(t, domainerrors.IsNotFound(err), "options go with the product, got %v", err)
}

func TestProductDelete_ArchivesOrderedProduct(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 50000, 5)
	h.checkoutOrder(t, f, 1)
	svc := NewProductService(h.products, h.variants, nil)

	require.NoError(t, svc.Delete(h.ctx, f.product.ID))

	product, err := h.products.GetByID(h.ctx, f.product.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ProductStatusArchived, product.Status)
	_, err = h.variants.GetByID(h.ctx, f.variant.ID)
	require.NoError(t, err, "order items keep resolving their option")
}

func TestVariantDelete_RemovesUnorderedOption(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 50000, 5)
	svc := NewProductVariantService(h.variants, h.inventory)

	require.NoError(t, svc.Delete(h.ctx, f.variant.ID))

	_, err := h.variants.GetByID(h.ctx, f.variant.ID)
	require.True(t, domainerrors.IsNotFound(err), "got %v", err)
	_, err = h.inventory.GetByVariantID(h.ctx, f.variant.ID)
	require.True(t, domainerrors.IsNotFound(err), "stock goes with the option, got %v", err)
}

func TestVariantDelete_HidesOrderedOption(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 50000, 5)
	h.checkoutOrder(t, f, 1)
	svc := NewProductVariantService(h.variants, h.inventory)

	require.NoError(t, svc.Delete(h.ctx, f.variant.ID))

	variant, err := h.variants.GetByID(h.ctx, f.variant.ID)
	require.NoError(t, err)
	require.False(t, variant.IsActive)
	require.Equal(t, f.variant.Title, variant.Title)
}

func TestVariantUpdate_KeepsTitleWhenOmitted(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 50000, 5)
	svc := NewProductVariantService(h.variants, h.inventory)

	hidden := false
	variant, err := svc.Update(h.ctx, f.variant.ID, UpdateVariantInput{IsActive: &hidden})
	require.NoError(t, err)
	require.False(t, variant.IsActive)
	require.Equal(t, f.variant.Title, variant.Title)
}
