package selector

import (
	"errors"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/hongy3025/ss/internal/parser"
)

var ErrAbort = errors.New("user aborted selection")

type Provider interface {
	Find(entries []parser.HostEntry, lastSelectedAlias string) (parser.HostEntry, error)
}

type FuzzyFinderProvider struct{}

func NewFuzzyFinderProvider() *FuzzyFinderProvider {
	return &FuzzyFinderProvider{}
}

func (p *FuzzyFinderProvider) Find(entries []parser.HostEntry, lastSelectedAlias string) (parser.HostEntry, error) {
	opts := []fuzzyfinder.Option{}

	if lastSelectedAlias != "" {
		opts = append(opts, fuzzyfinder.WithPreselected(func(i int) bool {
			return entries[i].Alias == lastSelectedAlias
		}))
	}

	idx, err := fuzzyfinder.Find(
		entries,
		func(i int) string { return entries[i].Display() },
		opts...,
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return parser.HostEntry{}, ErrAbort
		}
		return parser.HostEntry{}, err
	}
	return entries[idx], nil
}
