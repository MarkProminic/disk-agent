// Package keys implements the agent's API-key store: bcrypt-hashed keys
// persisted as a JSON file beside the config.
package keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/MarkProminic/disk-agent/internal/safepath"
)

// ErrNotFound is returned when no key has the requested id.
var ErrNotFound = errors.New("api key not found")

// bcrypt only reads the first 72 bytes of input, so both hashing and
// verification truncate there.
const bcryptMaxLen = 72

// Key is one API-key entity.
type Key struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	APIKeyHash string    `json:"api_key_hash"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsed   time.Time `json:"last_used"`
}

// Store is a mutex-guarded, file-backed key store.
type Store struct {
	mu   sync.Mutex
	path string
	data storeFile

	lastUsedFlush time.Time

	verified map[[sha256.Size]byte]int64
}

type storeFile struct {
	NextID int64  `json:"next_id"`
	Keys   []*Key `json:"keys"`
}

const lastUsedFlushInterval = time.Minute

// Open loads (or initializes) the store at path.
func Open(path string) (*Store, error) {
	clean, err := safepath.CleanAbs(path)
	if err != nil {
		return nil, err
	}
	s := &Store{
		path:     clean,
		data:     storeFile{NextID: 1},
		verified: map[[sha256.Size]byte]int64{},
	}

	raw, err := os.ReadFile(filepath.Clean(clean))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read key store %s: %w", clean, err)
	}
	if uerr := json.Unmarshal(raw, &s.data); uerr != nil {
		return nil, fmt.Errorf("parse key store %s: %w", clean, uerr)
	}
	if s.data.NextID < 1 {
		s.data.NextID = 1
	}
	return s, nil
}

// persist writes the store through safepath.WriteFile, 0600. Callers must
// hold s.mu.
func (s *Store) persist() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	if err := safepath.WriteFile(s.path, raw, 0o600); err != nil {
		return err
	}
	s.lastUsedFlush = time.Now()
	return nil
}

func truncateForBcrypt(key string) []byte {
	b := []byte(key)
	if len(b) > bcryptMaxLen {
		b = b[:bcryptMaxLen]
	}
	return b
}

// GenerateKeyString mints new key material: da_ + base64url(randomBytes).
func GenerateKeyString(keyLength int) (string, error) {
	raw := make([]byte, keyLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "da_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

// Create hashes apiKey and stores a new entity.
func (s *Store) Create(apiKey, name string, hashRounds int) (*Key, error) {
	hash, err := bcrypt.GenerateFromPassword(truncateForBcrypt(apiKey), hashRounds)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	k := &Key{
		ID:         s.data.NextID,
		Name:       name,
		APIKeyHash: string(hash),
		CreatedAt:  now,
		LastUsed:   now,
	}
	s.data.NextID++
	s.data.Keys = append(s.data.Keys, k)

	if err := s.persist(); err != nil {
		s.data.Keys = s.data.Keys[:len(s.data.Keys)-1]
		s.data.NextID--
		return nil, err
	}
	return k.clone(), nil
}

// Verify authenticates apiKey: a constant-time cache lookup for key material
// that has already proven itself this process lifetime, else a bcrypt
// comparison against every key. The bcrypt scan runs outside the store
// mutex. Returns the matching key or nil. lastUsed is bumped in memory on
// every hit and flushed to disk at most once per interval.
func (s *Store) Verify(apiKey string) (*Key, error) {
	candidate := truncateForBcrypt(apiKey)
	digest := sha256.Sum256(candidate)

	s.mu.Lock()
	if id, hit := s.verified[digest]; hit {
		k := s.find(id)
		if k != nil {
			match, err := s.recordUse(k)
			s.mu.Unlock()
			return match, err
		}
		delete(s.verified, digest)
	}
	all := make([]*Key, 0, len(s.data.Keys))
	for _, k := range s.data.Keys {
		all = append(all, k.clone())
	}
	s.mu.Unlock()

	for _, k := range all {
		if bcrypt.CompareHashAndPassword([]byte(k.APIKeyHash), candidate) != nil {
			continue
		}
		s.mu.Lock()
		live := s.find(k.ID)
		if live == nil {
			s.mu.Unlock()
			return nil, nil
		}
		s.verified[digest] = live.ID
		match, err := s.recordUse(live)
		s.mu.Unlock()
		return match, err
	}
	return nil, nil
}

// recordUse bumps a key's lastUsed (throttled persistence) and returns its
// clone. Callers must hold s.mu.
func (s *Store) recordUse(k *Key) (*Key, error) {
	k.LastUsed = time.Now()
	if time.Since(s.lastUsedFlush) > lastUsedFlushInterval {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	return k.clone(), nil
}

// List returns all keys, newest first.
func (s *Store) List() []*Key {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*Key, 0, len(s.data.Keys))
	for i := len(s.data.Keys) - 1; i >= 0; i-- {
		out = append(out, s.data.Keys[i].clone())
	}
	return out
}

// Delete permanently removes a key.
func (s *Store) Delete(id int64) (*Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.find(id)
	if k == nil {
		return nil, ErrNotFound
	}
	kept := s.data.Keys[:0]
	for _, other := range s.data.Keys {
		if other.ID != id {
			kept = append(kept, other)
		}
	}
	s.data.Keys = kept
	for digest, cachedID := range s.verified {
		if cachedID == id {
			delete(s.verified, digest)
		}
	}
	if err := s.persist(); err != nil {
		return nil, err
	}
	return k.clone(), nil
}

// find returns the live key with id. Callers must hold s.mu.
func (s *Store) find(id int64) *Key {
	for _, k := range s.data.Keys {
		if k.ID == id {
			return k
		}
	}
	return nil
}

func (k *Key) clone() *Key {
	c := *k
	return &c
}
