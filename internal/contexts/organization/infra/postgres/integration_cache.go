//go:build integration

package postgres

import "chnhcm.local/chnhcm/internal/contexts/organization/infra/history"

// ResetIntegrationCaches requires an idle, exclusively test-owned runtime.
// Corruption fixtures deliberately bypass immutability, so cached material
// must be discarded before observing those synthetic database changes.
func (s *Store) ResetIntegrationCaches() { s.listCache = nil; s.historyCatalog = history.Catalog{} }
