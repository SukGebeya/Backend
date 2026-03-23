package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
	domainerrors "github.com/nate/go-boilerplate/internal/errors"
	"github.com/nate/go-boilerplate/internal/repository"
)

var _ NoteManager = (*NoteService)(nil)

type NoteService struct {
	noteRepo repository.NoteRepository
}

func NewNoteService(noteRepo repository.NoteRepository) *NoteService {
	return &NoteService{noteRepo: noteRepo}
}

type CreateNoteInput struct {
	UserID  uuid.UUID
	Title   string
	Content string
}

type UpdateNoteInput struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Title   string
	Content string
}

func (s *NoteService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error) {
	return s.noteRepo.GetByID(ctx, id)
}

func (s *NoteService) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error) {
	return s.noteRepo.GetByIDAndUserID(ctx, id, userID)
}

func (s *NoteService) ListByUserID(ctx context.Context, userID uuid.UUID, opts domain.NoteListOptions) (*domain.PaginatedResult[domain.Note], error) {
	return s.noteRepo.ListByUserID(ctx, userID, opts)
}

func (s *NoteService) Create(ctx context.Context, input CreateNoteInput) (*domain.Note, error) {
	if input.Title == "" {
		return nil, domainerrors.Validation("title is required")
	}

	note := &domain.Note{
		ID:      uuid.New(),
		UserID:  input.UserID,
		Title:   input.Title,
		Content: input.Content,
	}

	return s.noteRepo.Create(ctx, note)
}

func (s *NoteService) Update(ctx context.Context, input UpdateNoteInput) (*domain.Note, error) {
	if input.Title == "" {
		return nil, domainerrors.Validation("title is required")
	}

	existing, err := s.noteRepo.GetByIDAndUserID(ctx, input.ID, input.UserID)
	if err != nil {
		return nil, err
	}

	if existing.IsDeleted() {
		return nil, domainerrors.BadRequest("cannot update deleted note")
	}

	existing.Title = input.Title
	existing.Content = input.Content

	return s.noteRepo.Update(ctx, existing)
}

func (s *NoteService) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	note, err := s.noteRepo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return err
	}

	if note.IsDeleted() {
		return domainerrors.BadRequest("note is already deleted")
	}

	return s.noteRepo.SoftDelete(ctx, id, userID)
}

func (s *NoteService) Restore(ctx context.Context, id, userID uuid.UUID) error {
	return s.noteRepo.Restore(ctx, id, userID)
}

func (s *NoteService) IsOwner(ctx context.Context, noteID, userID uuid.UUID) (bool, error) {
	note, err := s.noteRepo.GetByIDWithDeleted(ctx, noteID)
	if err != nil {
		return false, err
	}
	return note.UserID == userID, nil
}

func (s *NoteService) ValidateOwnership(ctx context.Context, noteID, userID uuid.UUID) error {
	isOwner, err := s.IsOwner(ctx, noteID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return domainerrors.Forbidden("you do not have access to this note")
	}
	return nil
}
