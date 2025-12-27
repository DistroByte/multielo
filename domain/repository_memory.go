package domain

import "strings"

type InMemoryPlayerRepo struct {
	byName  map[string]*Player
	ordered []*Player
}

func NewInMemoryPlayerRepo() *InMemoryPlayerRepo {
	return &InMemoryPlayerRepo{byName: make(map[string]*Player), ordered: make([]*Player, 0)}
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (r *InMemoryPlayerRepo) Get(name string) (*Player, bool) {
	p, ok := r.byName[normalize(name)]
	return p, ok
}

func (r *InMemoryPlayerRepo) Save(player *Player) error {
	key := normalize(player.Name())
	if _, exists := r.byName[key]; exists {
		return ErrPlayerAlreadyExists
	}
	r.byName[key] = player
	r.ordered = append(r.ordered, player)
	return nil
}

func (r *InMemoryPlayerRepo) Delete(name string) error {
	key := normalize(name)
	p, ok := r.byName[key]
	if !ok {
		return ErrPlayerNotFound
	}
	delete(r.byName, key)
	for i, existing := range r.ordered {
		if existing == p {
			r.ordered = append(r.ordered[:i], r.ordered[i+1:]...)
			break
		}
	}
	return nil
}

func (r *InMemoryPlayerRepo) List() []*Player {
	out := make([]*Player, len(r.ordered))
	copy(out, r.ordered)
	return out
}

func (r *InMemoryPlayerRepo) Count() int { return len(r.ordered) }

func (r *InMemoryPlayerRepo) Reset() {
	r.byName = make(map[string]*Player)
	r.ordered = make([]*Player, 0)
}

type InMemoryMatchRepo struct {
	matches []*Match
}

func NewInMemoryMatchRepo() *InMemoryMatchRepo {
	return &InMemoryMatchRepo{matches: make([]*Match, 0)}
}

func (r *InMemoryMatchRepo) Add(match *Match) error {
	r.matches = append(r.matches, match)
	return nil
}

func (r *InMemoryMatchRepo) List() []*Match {
	out := make([]*Match, len(r.matches))
	copy(out, r.matches)
	return out
}

func (r *InMemoryMatchRepo) Count() int { return len(r.matches) }

func (r *InMemoryMatchRepo) TrimOldest() bool {
	if len(r.matches) == 0 {
		return false
	}
	r.matches = r.matches[1:]
	return true
}

func (r *InMemoryMatchRepo) Reset() { r.matches = make([]*Match, 0) }
