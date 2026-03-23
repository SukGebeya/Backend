package handler

import (
	"net/http"

	"github.com/nate/go-boilerplate/internal/http/middleware"
	"github.com/nate/go-boilerplate/internal/service"
	"github.com/nate/go-boilerplate/pkg/response"
	"github.com/nate/go-boilerplate/pkg/validation"
)

type NoteHandler struct {
	noteService *service.NoteService
}

func NewNoteHandler(noteService *service.NoteService) *NoteHandler {
	return &NoteHandler{noteService: noteService}
}

type NoteResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Title     string  `json:"title" example:"My Note"`
	Content   string  `json:"content" example:"This is the content of my note"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	DeletedAt *string `json:"deleted_at,omitempty" example:"2024-01-15T10:30:00Z"`
}

type NoteListResponse struct {
	Notes      []NoteResponse `json:"notes"`
	Total      int64          `json:"total" example:"42"`
	Page       int            `json:"page" example:"1"`
	PerPage    int            `json:"per_page" example:"20"`
	TotalPages int            `json:"total_pages" example:"3"`
}

type CreateNoteRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=255" example:"My New Note"`
	Content string `json:"content" validate:"max=10000" example:"This is the content of my new note"`
}

type UpdateNoteRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=255" example:"Updated Note Title"`
	Content string `json:"content" validate:"max=10000" example:"This is the updated content"`
}

// ListNotes godoc
// @Summary List user's notes
// @Description Get a paginated list of notes belonging to the authenticated user. Supports soft-deleted notes with with_deleted parameter.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" minimum(1) default(1)
// @Param per_page query int false "Items per page" minimum(1) maximum(100) default(20)
// @Param with_deleted query bool false "Include soft-deleted notes" default(false)
// @Success 200 {object} NoteListResponse "List of notes"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes [get]
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	opts := parseNoteListOptions(r)

	result, err := h.noteService.ListByUserID(r.Context(), userID, opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	notes := make([]NoteResponse, len(result.Items))
	for i, n := range result.Items {
		notes[i] = *toNoteResponse(&n)
	}

	response.JSON(w, http.StatusOK, NoteListResponse{
		Notes:      notes,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// GetNote godoc
// @Summary Get a note by ID
// @Description Retrieve a specific note by its ID. User must own the note.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID" format(uuid)
// @Success 200 {object} NoteResponse "Note details"
// @Failure 400 {object} response.ErrorResponse "Invalid note ID format"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 404 {object} response.ErrorResponse "Note not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes/{id} [get]
func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	noteID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	note, err := h.noteService.GetByIDAndUserID(r.Context(), noteID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toNoteResponse(note))
}

// CreateNote godoc
// @Summary Create a new note
// @Description Create a new note for the authenticated user.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateNoteRequest true "Note details"
// @Success 201 {object} NoteResponse "Note created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes [post]
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	var req CreateNoteRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	note, err := h.noteService.Create(r.Context(), service.CreateNoteInput{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toNoteResponse(note))
}

// UpdateNote godoc
// @Summary Update a note
// @Description Update an existing note. User must own the note.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID" format(uuid)
// @Param request body UpdateNoteRequest true "Updated note details"
// @Success 200 {object} NoteResponse "Note updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, note ID, or validation error"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 404 {object} response.ErrorResponse "Note not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes/{id} [put]
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	noteID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateNoteRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	note, err := h.noteService.Update(r.Context(), service.UpdateNoteInput{
		ID:      noteID,
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toNoteResponse(note))
}

// DeleteNote godoc
// @Summary Soft delete a note
// @Description Soft delete a note (move to trash). User must own the note.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID" format(uuid)
// @Success 204 "Note deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid note ID format"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 404 {object} response.ErrorResponse "Note not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes/{id} [delete]
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	noteID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.noteService.SoftDelete(r.Context(), noteID, userID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RestoreNote godoc
// @Summary Restore a soft-deleted note
// @Description Restore a soft-deleted note from trash. User must own the note.
// @Tags notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID" format(uuid)
// @Success 204 "Note restored successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid note ID format"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 404 {object} response.ErrorResponse "Note not found or not deleted"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/notes/{id}/restore [post]
func (h *NoteHandler) Restore(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	noteID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.noteService.Restore(r.Context(), noteID, userID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
