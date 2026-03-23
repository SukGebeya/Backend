package domain

type PaginatedResult[T any] struct {
	Items      []T
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
}

type ListOptions struct {
	Page    int
	PerPage int
}

func DefaultListOptions() ListOptions {
	return ListOptions{
		Page:    1,
		PerPage: 20,
	}
}

func (o ListOptions) Offset() int {
	return (o.Page - 1) * o.PerPage
}

func (o ListOptions) Limit() int {
	return o.PerPage
}
