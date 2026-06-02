package selector

import (
	"errors"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/hongy3025/ss/internal/parser"
)

var ErrAbort = errors.New("user aborted selection")

type Provider interface {
	Find(entries []parser.HostEntry) (parser.HostEntry, error)
}

type FuzzyFinderProvider struct{}

func NewFuzzyFinderProvider() *FuzzyFinderProvider {
	return &FuzzyFinderProvider{}
}

func (p *FuzzyFinderProvider) Find(entries []parser.HostEntry) (parser.HostEntry, error) {
	idx, err := fuzzyfinder.Find(
		entries,
		func(i int) string { return entries[i].Display() },
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return parser.HostEntry{}, ErrAbort
		}
		return parser.HostEntry{}, err
	}
	return entries[idx], nil
}
