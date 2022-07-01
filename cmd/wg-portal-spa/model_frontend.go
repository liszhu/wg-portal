package main

type PagedResponse[T any] struct {
	Records     []T
	MoreRecords bool
}
