// Package cachewithouthits models endpoint metadata reuse within one snapshot.
package cachewithouthits

import "time"

// Metadata is stable for an endpoint during one Resolver's lifetime.
type Metadata struct {
	Owner  string
	Region string
}

// Observation is an endpoint event awaiting ownership metadata.
type Observation struct {
	IP         string
	ObservedAt time.Time
}

// EnrichedObservation combines an observation with endpoint metadata.
type EnrichedObservation struct {
	Observation Observation
	Metadata    Metadata
}

// Loader retrieves metadata by endpoint IP.
type Loader interface {
	LoadEndpoint(ip string) (Metadata, error)
}

type cacheKey struct {
	IP                 string
	ObservedAtUnixNano int64
}

// Resolver reuses loaded metadata for matching cache keys.
type Resolver struct {
	loader Loader
	cache  map[cacheKey]Metadata
}

// NewResolver constructs an empty resolver for one metadata snapshot.
func NewResolver(loader Loader) *Resolver {
	return &Resolver{
		loader: loader,
		cache:  make(map[cacheKey]Metadata),
	}
}

// Resolve enriches observations in input order.
func (r *Resolver) Resolve(observations []Observation) ([]EnrichedObservation, error) {
	result := make([]EnrichedObservation, len(observations))
	for i, observation := range observations {
		key := cacheKey{
			IP:                 observation.IP,
			ObservedAtUnixNano: observation.ObservedAt.UnixNano(),
		}
		metadata, ok := r.cache[key]
		if !ok {
			var err error
			metadata, err = r.loader.LoadEndpoint(observation.IP)
			if err != nil {
				return nil, err
			}
			r.cache[key] = metadata
		}
		result[i] = EnrichedObservation{
			Observation: observation,
			Metadata:    metadata,
		}
	}
	return result, nil
}

// EntryCount returns the number of retained cache entries.
func (r *Resolver) EntryCount() int {
	return len(r.cache)
}
