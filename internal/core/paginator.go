package core

import (
	"fmt"
	"sort"
)

var ErrNoMorePage = fmt.Errorf("no more page")

const PageSizeAll = 0

type Paginator[T any] interface {
	Size(size int) Paginator[T]
	Sort(less func(a, b T) bool) Paginator[T]
	Paginate(offset int) ([]T, error)
}

type inMemoryPaginator[T any] struct {
	pageSize int
	elements []T
}

func NewInMemoryPaginator[T any](elements []T) *inMemoryPaginator[T] {
	return &inMemoryPaginator[T]{
		pageSize: PageSizeAll,
		elements: elements,
	}
}

// Size overrides the default size (PageSizeAll) of the current Paginator instance.
func (p *inMemoryPaginator[T]) Size(size int) Paginator[T] {
	p.pageSize = size
	return p
}

// Sort sorts the internal elements of the current Paginator instance.
func (p *inMemoryPaginator[T]) Sort(less func(a, b T) bool) Paginator[T] {
	sort.Slice(p.elements, func(i, j int) bool {
		return less(p.elements[i], p.elements[j])
	})
	return p
}

func (p *inMemoryPaginator[T]) Paginate(offset int) ([]T, error) {
	if p.pageSize == PageSizeAll {
		return p.elements, nil // no paging
	}

	if offset >= len(p.elements) {
		return nil, ErrNoMorePage
	}

	end := offset + p.pageSize
	if end > len(p.elements) {
		end = len(p.elements)
	}

	return p.elements[offset:end], nil
}
